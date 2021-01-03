// Package app manages main application server.
package app

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	rice "github.com/GeertJohan/go.rice"
	"github.com/dgrijalva/jwt-go"
	"github.com/dustin/go-humanize"
	"github.com/fsnotify/fsnotify"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/prologic/tube/media"
	"github.com/prologic/tube/models"
	"github.com/prologic/tube/utils"
	"github.com/renstrom/shortuuid"
	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

//go:generate rice embed-go

// App represents main application.
type App struct {
	Config    *Config
	Library   *media.Library
	Store     Store
	Watcher   *fsnotify.Watcher
	Templates *templateStore
	Feed      []byte
	Listener  net.Listener
	Router    *mux.Router
	DataBase  *gorm.DB
}

// NewApp returns a new instance of App from Config.
func NewApp(cfg *Config) (*App, error) {
	if cfg == nil {
		cfg = DefaultConfig()
	}
	app := &App{
		Config: cfg,
	}
	// Setup Library
	app.Library = media.NewLibrary()
	// Setup Store
	store, err := NewBitcaskStore(cfg.Server.StorePath)
	if err != nil {
		err := fmt.Errorf("error opening store %s: %w", cfg.Server.StorePath, err)
		return nil, err
	}
	app.Store = store
	// Setup Watcher
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	app.Watcher = w
	// Setup Listener
	ln, err := newListener(cfg.Server)
	if err != nil {
		return nil, err
	}
	app.Listener = ln

	// Templates
	box := rice.MustFindBox("../templates")

	app.Templates = newTemplateStore("base")

	templateFuncs := map[string]interface{}{
		"bytes": func(size int64) string { return humanize.Bytes(uint64(size)) },
	}

	indexTemplate := template.New("index").Funcs(templateFuncs)
	template.Must(indexTemplate.Parse(box.MustString("index.html")))
	template.Must(indexTemplate.Parse(box.MustString("base.html")))
	app.Templates.Add("index", indexTemplate)

	uploadTemplate := template.New("upload").Funcs(templateFuncs)
	template.Must(uploadTemplate.Parse(box.MustString("upload.html")))
	template.Must(uploadTemplate.Parse(box.MustString("base.html")))
	app.Templates.Add("upload", uploadTemplate)

	// Setup Router
	router := mux.NewRouter().StrictSlash(true)
	router.HandleFunc("/", app.indexHandler).Methods("GET", "OPTIONS")
	router.HandleFunc("/v/list", app.listVideosHandler).Methods("GET")
	router.HandleFunc("/v/{id}.mp4", app.getVideoHandler).Methods("GET")
	router.HandleFunc("/v/{id}", app.getVideoInfoHandler).Methods("GET", "OPTIONS")
	router.HandleFunc("/user/{id}", app.getProfileHandler).Methods("GET", "OPTIONS")
	router.HandleFunc("/user/{id}/video", app.getUserVideosHandler).Methods("GET", "OPTIONS")
	router.HandleFunc("/feed.xml", app.rssHandler).Methods("GET")

	router.HandleFunc("/auth/signup", app.apiCreateUserHandler).Methods("POST", "OPTIONS")
	router.HandleFunc("/auth/login", app.loginHandler).Methods("POST", "OPTIONS")

	api := router.PathPrefix("/api").Subrouter()
	api.Use(app.jwtVerify)
	api.HandleFunc("/video", app.apiUploadVideoHandler).Methods("POST", "OPTIONS")
	api.HandleFunc("/video/{id}", app.apiDeleteVideoHandler).Methods("DELETE", "OPTIONS")
	api.HandleFunc("/video/{id}/comments", app.apiGetVideoCommentsHandler).Methods("GET", "OPTIONS")
	api.HandleFunc("/like/{id}", app.apiLikeHandler).Methods("POST", "DELETE", "OPTIONS")
	api.HandleFunc("/like/{id}", app.apiCheckLiked).Methods("GET")
	api.HandleFunc("/dislike/{id}", app.apiDislikeHandler).Methods("POST", "DELETE", "OPTIONS")
	api.HandleFunc("/comment", app.apiCreateCommentHandler).Methods("POST", "OPTIONS")
	api.HandleFunc("/comment/{id}", app.apiGetCommentHandler).Methods("GET", "OPTIONS")
	api.HandleFunc("/comment/{id}", app.apiDeleteCommentHandler).Methods("DELETE")

	admin := router.PathPrefix("/admin").Subrouter()
	admin.Use(app.jwtVerifyAdmin)
	admin.HandleFunc("/user", app.apiAdminGetUsersHandler).Methods("GET", "OPTIONS")
	admin.HandleFunc("/user/chart", app.adminGetUserChartHandler).Methods("GET", "OPTIONS")
	admin.HandleFunc("/user/{id}", app.adminDeleteUserHandler).Methods("DELETE", "OPTIONS")
	admin.HandleFunc("/video", app.apiAdminGetVideosHandler).Methods("GET", "OPTIONS")

	// Static assets handler
	// staticFs := http.FileServer(http.Dir("./static"))
	// router.PathPrefix("/static/").
	// 	Handler(http.StripPrefix("/static/", staticFs)).
	// 	Methods("GET")

	// Uploads static handler
	uploadsFs := http.FileServer(http.Dir("./uploads"))
	router.PathPrefix("/uploads/").
		Handler(http.StripPrefix("/uploads/", uploadsFs)).
		Methods("GET")

	cors := handlers.CORS(
		handlers.AllowedHeaders([]string{
			"X-Requested-With",
			"Content-Type",
			"Authorization",
		}),
		handlers.AllowedMethods([]string{
			"GET", "POST", "PUT", "DELETE", "HEAD", "OPTIONS",
		}),
		handlers.AllowedOrigins([]string{"*"}),
		handlers.AllowCredentials(),
	)

	router.Use(cors)

	app.Router = router
	return app, nil
}

//ConnectDB function: Make database connection
func ConnectDB() (*gorm.DB, error) {
	dsn := "tubeadmin:drowssap@tcp(127.0.0.1:3306)/nontube?charset=utf8mb4&parseTime=True&loc=Local"
	return gorm.Open(mysql.Open(dsn), &gorm.Config{})
}

// Run imports the library and starts server.
func (app *App) Run() error {
	var err error

	app.DataBase, err = ConnectDB()
	if err != nil {
		return err
	}

	for _, pc := range app.Config.Library {
		p := &media.Path{
			Path:   pc.Path,
			Prefix: pc.Prefix,
		}
		err = app.Library.AddPath(p)
		if err != nil {
			return err
		}
		err = app.Library.Import(p)
		if err != nil {
			return err
		}
		app.Watcher.Add(p.Path)
	}

	if err := os.MkdirAll(app.Config.Server.UploadPath, 0755); err != nil {
		return fmt.Errorf(
			"Error creating upload path %s: %w",
			app.Config.Server.UploadPath, err,
		)
	}

	buildFeed(app)
	go startWatcher(app)
	return http.Serve(app.Listener, app.Router)
}

func (app *App) render(name string, w http.ResponseWriter, ctx interface{}) {
	buf, err := app.Templates.Exec(name, ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	_, err = buf.WriteTo(w)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// HTTP handler for /
func (app *App) indexHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("/")
	pl := app.Library.Playlist()
	if len(pl) > 0 {
		http.Redirect(w, r, fmt.Sprintf("/v/%s?%s", pl[0].ID, r.URL.RawQuery), 302)
	} else {
		sort := strings.ToLower(r.URL.Query().Get("sort"))
		quality := strings.ToLower(r.URL.Query().Get("quality"))
		ctx := &struct {
			Sort     string
			Quality  string
			Playing  *media.Video
			Playlist media.Playlist
		}{
			Sort:     sort,
			Quality:  quality,
			Playing:  &media.Video{ID: ""},
			Playlist: app.Library.Playlist(),
		}

		app.render("index", w, ctx)
	}
}

// HTTP handler for /feed.xml
func (app *App) rssHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "public, max-age=7776000")
	w.Header().Set("Content-Type", "text/xml")
	w.Write(app.Feed)
}

// HTTP handler for /auth/signup
func (app *App) apiCreateUserHandler(w http.ResponseWriter, r *http.Request) {
	user := &models.User{}
	json.NewDecoder(r.Body).Decode(user)

	pass, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "Password encryption failed!", http.StatusInternalServerError)
		log.Error(err)
		return
	}

	user.Password = string(pass)

	res := app.DataBase.Create(&user)
	if res.Error != nil {
		http.Error(w, "Failed to create user!", http.StatusInternalServerError)
		log.Error(res.Error)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

// HTTP handler for /auth/login
func (app *App) loginHandler(w http.ResponseWriter, r *http.Request) {
	log.Info("[POST] /auth/login")
	user := &models.User{}
	err := json.NewDecoder(r.Body).Decode(user)
	if err != nil {
		http.Error(w, "Login failed", http.StatusBadRequest)
		log.Error(err)
		return
	}

	resp, err := app.findUser(user.Name, user.Password)
	if err != nil {
		http.Error(w, "Login failed", http.StatusBadRequest)
		log.Error(err)
	} else {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}
}

func (app *App) findUser(name, password string) (map[string]interface{}, error) {
	user := &models.User{}
	err := app.DataBase.Where("name = ?", name).First(user).Error
	if err != nil {
		return nil, fmt.Errorf("User name not found")
	}

	expiresAt := time.Now().Add(time.Minute * 100000).Unix()
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	// Password does not match!
	if err != nil && err == bcrypt.ErrMismatchedHashAndPassword {
		return nil, fmt.Errorf("Invalid login credentials. Please try again")
	}

	tk := &models.UserClaims{
		UserID: user.ID,
		Name:   user.Name,
		Email:  user.Email,
		Password: user.Password,
		IsAdmin: user.IsAdmin,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: expiresAt,
		},
	}

	token := jwt.NewWithClaims(jwt.GetSigningMethod("HS256"), tk)
	tokenString, error := token.SignedString([]byte("secret"))
	if error != nil {
		fmt.Println(error)
	}

	var resp = map[string]interface{}{"status": false, "message": "Logged in successfully!"}
	resp["token"] = tokenString // Store the token in the response
	resp["user"] = user
	return resp, nil
}

// MIDDLEWARE FOR USER AUTHENTICATION
func (app *App) jwtVerify(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get("Authorization")
		header = strings.TrimSpace(header)
		if header == "" {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("Token is not provided"))
			return
		}

		tk := &models.UserClaims{}
		_, err := jwt.ParseWithClaims(header, tk, func(token *jwt.Token) (interface{}, error) {
			return []byte("secret"), nil
		})

		if err == nil {
			ctx := context.WithValue(r.Context(), "userID", tk.UserID)
			next.ServeHTTP(w, r.WithContext(ctx))
		} else {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("Authentication error"))
		}
	})
}

// MIDDLEWARE FOR ADMIN AUTHENTICATION
func (app *App) jwtVerifyAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get("Authorization")
		header = strings.TrimSpace(header)
		if header == "" {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("Token is not provided"))
			return
		}

		tk := &models.UserClaims{}
		_, err := jwt.ParseWithClaims(header, tk, func(token *jwt.Token) (interface{}, error) {
			return []byte("secret"), nil
		})

		if err == nil {
			if tk.IsAdmin {
				ctx := context.WithValue(r.Context(), "userID", tk.UserID)
				next.ServeHTTP(w, r.WithContext(ctx))
			} else {
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte("You don't have admin priveleges!"))
			}
		} else {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("Authentication error"))
		}
	})
}


// HTTP handler for /api/upload
func (app *App) apiUploadVideoHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseMultipartForm(app.Config.Server.MaxUploadSize)

	// GET VIDEO FROM REQUEST
	file, handler, err := r.FormFile("video")
	if err != nil {
		log.Error(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer file.Close()

	// CREATE VIDEO OBJECT
	vid := &models.Video{}
	vid.Title = r.FormValue("title")
	vid.Description = r.FormValue("description")
	uid_ctx := r.Context().Value("userID")
	vid.UserID = uid_ctx.(uint)

	// CREATE TEMP COPY OF VIDEO
	tempCopy, err := ioutil.TempFile(
		app.Config.Server.UploadPath,
		fmt.Sprintf("tube-upload-*%s", filepath.Ext(handler.Filename)),
	)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		log.Error(err, w)
		return
	}

	_, err = io.Copy(tempCopy, file)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		log.Error(err, w)
		return
	}

	// GENERATE RANDOM IDNTIFIER
	uniqueName := shortuuid.New()
	vid.URL = filepath.Join(app.Config.Server.UploadPath, fmt.Sprintf("%s.mp4", uniqueName))
	vid.ThumbnailURL = filepath.Join(app.Config.Server.UploadPath, fmt.Sprintf("%s.jpg", uniqueName))

	res := app.DataBase.Create(&vid)
	if res.Error != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		log.Error(res.Error)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(vid)
	log.Info(fmt.Sprintf("New upload: Id=%d; Title: \"%s\"", vid.ID, vid.Title))

	defer app.processVideo(vid, uniqueName, tempCopy)
}

func (app *App) processVideo(video *models.Video, uniqueName string, tempCopy *os.File) {
	transcodeFile, err := ioutil.TempFile(
		app.Config.Server.UploadPath,
		fmt.Sprintf("tube-transcode-*.mp4"),
	)
	tempThumb := fmt.Sprintf("%s.jpg", strings.TrimSuffix(transcodeFile.Name(), ".mp4"))
	destThumb := filepath.Join(app.Config.Server.UploadPath, fmt.Sprintf("%s.jpg", uniqueName))
	destVid := filepath.Join(app.Config.Server.UploadPath, fmt.Sprintf("%s.mp4", uniqueName))

	if err := utils.RunCmd(
		app.Config.Transcoder.Timeout,
		"ffmpeg", "-y", "-i",
		tempCopy.Name(),
		"-vcodec", "h264", "-acodec", "aac",
		"-strict", "-2", "-loglevel", "quiet",
		"-metadata", fmt.Sprintf("title=%s", video.Title),
		"-metadata", fmt.Sprintf("comment=%s", video.Description),
		transcodeFile.Name(),
	); err != nil {
		log.Error(err)
		return
	}

	video.Duration, err = getVideoDuration(transcodeFile.Name())
	app.DataBase.Save(&video)
	if err != nil {
		log.Error(err)
		return
	}

	err = utils.RunCmd(app.Config.Thumbnailer.Timeout,
		"mt", "-b", "-s", "-n", "1",
		transcodeFile.Name(),
	)
	if err != nil {
		log.Error(err)
		return
	}

	if err := os.Rename(tempThumb, destThumb); err != nil {
		log.Error(err)
		return
	}
	if err := os.Rename(transcodeFile.Name(), destVid); err != nil {
		log.Error(err)
		return
	}
	log.Info("Video processed!")
	defer os.Remove(tempCopy.Name())
}

func getVideoDuration(filename string) (int, error) {
	cmd := fmt.Sprintf("ffmpeg -i %s 2>&1 | grep Duration | awk '{print $2}'", filename)
	out, err := exec.Command("bash", "-c", cmd).Output()
	if err != nil {
		return -1, err
	}
	tmstr := strings.TrimSpace(string(out[:]))
	tmstr = strings.Trim(tmstr, ",")

	tm, err := time.Parse("15:04:05.00", tmstr)
	dur := tm.Second()
	return dur, nil
}

func (app *App) apiDeleteVideoHandler(w http.ResponseWriter, r *http.Request) {
	uid_ctx := r.Context().Value("userID")
	uid := uid_ctx.(uint)

	id := mux.Vars(r)["id"]
	log.Info(fmt.Sprintf("Deleting a video; id = %s", id))

	video := &models.Video{}
	app.DataBase.Find(video, id)

	if video.UserID != uid {
		http.Error(w, "You are not the owner of this video", http.StatusForbidden)
		log.Error("Deletion not permitted")
		return
	}

	if err := os.Remove(video.URL); err != nil {
		http.Error(w, "Error", http.StatusInternalServerError)
		log.Error(err)
		return
	}
	if err := os.Remove(video.ThumbnailURL); err != nil {
		http.Error(w, "Error", http.StatusInternalServerError)
		log.Error(err)
		return
	}

	app.DataBase.Delete(&video)
}

// HTTP handler for /v/list
func (app *App) listVideosHandler(w http.ResponseWriter, r *http.Request) {
	var videos []models.Video
	app.DataBase.Preload("User").Find(&videos)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(videos)
}

// HTTP handler for /v/id.mp4
func (app *App) getVideoHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	log.Printf("/v/%s.mp4", id)

	video := &models.Video{}
	app.DataBase.First(video, id)

	if video.ID > 0 {
		_, filename := path.Split(video.URL)
		disposition := `attachment; filename="` + filename + `"`
		w.Header().Set("Content-Disposition", disposition)
		w.Header().Set("Content-Type", "video/mp4")
		http.ServeFile(w, r, video.URL)
	} else {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("Video not found"))
	}
}

func (app *App) incrementViews(vid *models.Video) {
	log.Info("Incrementing views")
	vid.Views++
	app.DataBase.Save(&vid)
}

// HTTP handler for /v/id
func (app *App) getVideoInfoHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	log.Printf("/v/%s", id)

	video := &models.Video{}
	app.DataBase.First(video, id)

	if video.ID > 0 {
		app.DataBase.First(&video.User, video.UserID)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(video)
	} else {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("Video not found"))
	}

	defer app.incrementViews(video)
}

// HTTP handler for [GET] /user/id
func (app *App) getProfileHandler(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	user := &models.User{}
	app.DataBase.Find(user, id);
	if user.ID <= 0 {
		http.Error(w, "User not found", http.StatusNotFound)
		log.Info("User not found")
		return
	}

	user.Password = ""
	json.NewEncoder(w).Encode(user)
}

// HTTP handler for [GET] /user/id/video
func (app *App) getUserVideosHandler(w http.ResponseWriter, r *http.Request) {
	uid := mux.Vars(r)["id"]

	user := &models.User{}
	app.DataBase.Find(user, uid);
	if user.ID <= 0 {
		http.Error(w, "User not found", http.StatusBadRequest)
		log.Info("User not found")
		return
	}

	var videos []models.Video
	app.DataBase.Where("user_id = ?", uid).Find(&videos)

	for i := range videos {
		videos[i].User = *user
	}

	json.NewEncoder(w).Encode(videos)
}



func (app *App) apiLikeHandler(w http.ResponseWriter, r *http.Request) {
	vID := mux.Vars(r)["id"]
	uidCtx := r.Context().Value("userID")

	like := &models.Like{}
	app.DataBase.Table("likes").Where("uid = ? AND v_id = ?", uidCtx, vID).First(like)

	video := &models.Video{}
	app.DataBase.First(video, vID)
	if video.ID <= 0 {
		http.Error(w, "Video not found", http.StatusNotFound)
		log.Info("Video not found")
		return
	}

	if r.Method == http.MethodPost {
		// IF LIKE FOUND AND IT IS DISLIKE
		if like.ID > 0 && like.IsDislike {
			// change to like and save
			like.IsDislike = false
			video.Dislikes--
			video.Likes++
			app.DataBase.Save(like)
			app.DataBase.Save(video)
		} else if like.ID > 0 && !like.IsDislike {
			// already exits
		} else {
			// new like
			video.Likes++
			like.UID = uidCtx.(uint)
			like.VID = video.ID
			app.DataBase.Save(like)
			app.DataBase.Save(video)
		}
	} else if r.Method == http.MethodDelete {
		// IF LIKE FOUND AND IT IS NOT DISLIKE
		if like.ID > 0 && like.IsDislike == false {
			video.Likes--
			app.DataBase.Delete(like)
			app.DataBase.Save(video)
		} else {
			http.Error(w, "Bad request", http.StatusBadRequest)
			log.Info("Bad request")
		}
	}
}

func (app *App) apiCheckLiked(w http.ResponseWriter, r *http.Request) {
	vID := mux.Vars(r)["id"]
	uidCtx := r.Context().Value("userID")

	like := &models.Like{}
	app.DataBase.Table("likes").Where("uid = ? AND v_id = ?", uidCtx, vID).First(like)

	video := &models.Video{}
	app.DataBase.First(video, vID)
	if video.ID <= 0 {
		http.Error(w, "Video not found", http.StatusNotFound)
		log.Info("Video not found")
		return
	}

	resp := map[string]bool{
		"liked":    (like.ID > 0 && !like.IsDislike),
		"disliked": (like.ID > 0 && like.IsDislike),
	}
	json.NewEncoder(w).Encode(resp)
}

func (app *App) apiDislikeHandler(w http.ResponseWriter, r *http.Request) {
	vID := mux.Vars(r)["id"]
	uidCtx := r.Context().Value("userID")

	like := &models.Like {}
	app.DataBase.Table("likes").Where("uid = ? AND v_id = ?", uidCtx, vID).First(like)

	video := &models.Video{}
	app.DataBase.First(video, vID)
	if video.ID <= 0 {
		http.Error(w, "Video not found", http.StatusBadRequest)
		log.Info("Video not found")
		return
	}

	if r.Method == http.MethodPost {
		// IF LIKE FOUND AND IT IS NOT DISLIKE
		if like.ID > 0 && like.IsDislike == false {
			// change to like and save
			like.IsDislike = false
			video.Dislikes++
			video.Likes--
			app.DataBase.Save(like)
			app.DataBase.Save(video)
		} else if like.ID > 0 && like.IsDislike {
			// already exits
		} else {
			// new dislike
			video.Dislikes++
			like.IsDislike = true
			like.UID = uidCtx.(uint)
			like.VID = video.ID
			app.DataBase.Save(like)
			app.DataBase.Save(video)
		}
	} else if r.Method == http.MethodDelete {
		// IF LIKE FOUND AND IT IS DISLIKE
		if like.ID > 0 && like.IsDislike {
			video.Dislikes--
			app.DataBase.Delete(like)
			app.DataBase.Save(video)
		} else {
			http.Error(w, "Bad request", http.StatusBadRequest)
			log.Info("Bad request")
		}
	}
}

// HTTP handler for [POST] /api/comment/
func (app *App) apiCreateCommentHandler(w http.ResponseWriter, r *http.Request) {
	uidCtx := r.Context().Value("userID")


	comment := &models.Comment{}
	err := json.NewDecoder(r.Body).Decode(comment)
	if (err != nil) {
		http.Error(w, "Bad request", http.StatusBadRequest)
		log.Info(err)
		return
	}
	comment.UserID = uidCtx.(uint)

	log.Info(fmt.Sprintf("Creating comment %s", comment.Text))

	// CHECK VIDEO EXISTS
	video := &models.Video{}
	app.DataBase.Find(video, comment.VideoID)
	if video.ID <= 0 {
		http.Error(w, "Video not found", http.StatusBadRequest)
		log.Info("Video not found")
		return
	}

	// CHECK REFERENCED COMMENT EXISTS
	if (comment.ReplyTo.Int64 > 0) {
		refcomm := &models.Comment{}
		app.DataBase.Find(refcomm, comment.ReplyTo);
		if refcomm.ID <= 0 {
			http.Error(w, "Refrenced comment not found", http.StatusBadRequest)
			log.Info("Refrenced comment not found")
			return
		}
	} else {
		comment.ReplyTo.Valid = false
	}
	res := app.DataBase.Save(comment)
	
	if res.Error != nil {
		http.Error(w, res.Error.Error(), http.StatusInternalServerError)
		log.Error(res.Error)
		return
	}
	json.NewEncoder(w).Encode(comment)
}

func (app *App) apiDeleteCommentHandler(w http.ResponseWriter, r *http.Request) {
	commID := mux.Vars(r)["id"]

	comment := &models.Comment{}
	app.DataBase.Find(comment, commID)
	if comment.ID <= 0 {
		http.Error(w, "Comment not found", http.StatusBadRequest)
		log.Info("Comment not found")
		return
	}

	app.DataBase.Delete(&comment)

	json.NewEncoder(w).Encode(comment)
}

// HTTP handler for [GET] /api/comment/{id}
func (app *App) apiGetCommentHandler(w http.ResponseWriter, r *http.Request) {
	commID := mux.Vars(r)["id"]

	comment := &models.Comment{}
	app.DataBase.Find(comment, commID)
	if comment.ID <= 0 {
		http.Error(w, "Comment not found", http.StatusBadRequest)
		log.Info("Comment not found")
		return
	}

	res := app.DataBase.
		Table("comments").
		Where("reply_to = ? AND deleted_at IS NULL", comment.ID).
		Scan(&comment.Replies)
	if res.Error != nil {
		http.Error(w, res.Error.Error(), http.StatusInternalServerError)
		log.Error(res.Error)
		return
	}

	comment.ReplyCount = len(comment.Replies)
	for i := range comment.Replies {
		app.DataBase.Find(&comment.Replies[i].User, comment.Replies[i].UserID)
		app.DataBase.
			Raw("SELECT COUNT(*) FROM comments WHERE reply_to = ? AND deleted_at IS NULL;", comment.Replies[i].ID).
			Scan(&comment.Replies[i].ReplyCount)
	}

	json.NewEncoder(w).Encode(comment);
}

// HTTP handler for [GET] /api/video/{id}/comments
// This thing is really slooow
func (app *App) apiGetVideoCommentsHandler(w http.ResponseWriter, r *http.Request) {
	vID := mux.Vars(r)["id"]

	comments := []models.Comment{}

	res := app.DataBase.
		Table("comments").
		Where("video_id = ? AND reply_to IS NULL AND deleted_at IS NULL", vID).
		Scan(&comments)
	if res.Error != nil {
		http.Error(w, res.Error.Error(), http.StatusInternalServerError)
		log.Error(res.Error)
		return
	}

	for i := range comments {
		app.DataBase.Find(&comments[i].User, comments[i].UserID)
		app.DataBase.
			Raw("SELECT COUNT(*) FROM comments WHERE reply_to = ? AND deleted_at IS NULL;", comments[i].ID).
			Scan(&comments[i].ReplyCount)
	}

	json.NewEncoder(w).Encode(comments)
}


// HTTP handler for [GET] /admin/user
func (app *App) apiAdminGetUsersHandler(w http.ResponseWriter, r *http.Request) {
	offset, limit := getOffsetAndLimit(r.URL.Query())
	users := []models.User{}
	app.DataBase.Limit(limit).Offset(offset).
				 Find(&users)

	var total int
	app.DataBase.
		Raw("SELECT COUNT(*) FROM users WHERE deleted_at IS NULL").
		Scan(&total)

	var resp = map[string]interface{}{
		"total": total, 
		"offset": offset,
		"count": len(users),
		"users": users,
	}
	json.NewEncoder(w).Encode(resp);
}

func getOffsetAndLimit(query url.Values) (int, int) {
	var offset, limit int
	offsets, ok := query["offset"]
	if !ok || len(offsets[0]) < 1 {
		offset = 0
	} else {
		offset, _ = strconv.Atoi(offsets[0])
	}
	limits, ok := query["limit"]
	if !ok || len(limits[0]) < 1 {
		limit = 10
	} else {
		limit, _ = strconv.Atoi(limits[0])
	}
	return offset, limit
}

// HTTP handler for [DELETE] /admin/user/id
func (app *App) adminDeleteUserHandler(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	log.Info(fmt.Sprintf("Deleting a user (as admin); id = %s", id))

	user := &models.User{}
	app.DataBase.Find(user, id)

	app.DataBase.Delete(&user)
}

// HTTP handler for [DELETE] /admin/video/id
func (app *App) adminDeleteVideoHandler(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	log.Info(fmt.Sprintf("Deleting a video (as admin); id = %s", id))

	video := &models.Video{}
	app.DataBase.Find(video, id)

	if err := os.Remove(video.URL); err != nil {
		http.Error(w, "Error", http.StatusInternalServerError)
		log.Error(err)
		return
	}
	if err := os.Remove(video.ThumbnailURL); err != nil {
		http.Error(w, "Error", http.StatusInternalServerError)
		log.Error(err)
		return
	}

	app.DataBase.Delete(&video)

}

// HTTP handler for [GET] /admin/user/chart
func (app *App) adminGetUserChartHandler(w http.ResponseWriter, r *http.Request) {
	q := "SELECT *, " +
		 "(SELECT IFNULL(SUM(views), 0) FROM videos WHERE user_id = users.id) AS total_views, " +
		 "(SELECT COUNT(*) FROM videos WHERE user_id = users.id) AS num_videos " +
		 "FROM users WHERE deleted_at IS NULL ORDER BY total_views DESC LIMIT 10;"

	users := []models.UserStat{}
	res := app.DataBase.Raw(q).Scan(&users)
	if res.Error != nil {
		log.Error(res.Error)
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(users)
}

// HTTP handler for [GET] /admin/video
func (app *App) apiAdminGetVideosHandler(w http.ResponseWriter, r *http.Request) {
	offset, limit := getOffsetAndLimit(r.URL.Query())
	videos := []models.Video{}
	app.DataBase.Limit(limit).Offset(offset).
				 Preload("User").
				 Find(&videos)

	var total int
	app.DataBase.
		Raw("SELECT COUNT(*) FROM videos WHERE deleted_at IS NULL").
		Scan(&total)

	var resp = map[string]interface{}{
		"total": total, 
		"offset": offset,
		"count": len(videos),
		"videos": videos,
	}
	json.NewEncoder(w).Encode(resp);
}


// TODO: Make this a background job
// Resize for lower quality options
// for size, suffix := range app.Config.Transcoder.Sizes {
// 	log.
// 		WithField("size", size).
// 		WithField("vf", filepath.Base(vf)).
// 		Info("resizing video for lower quality playback")
// 	sf := fmt.Sprintf(
// 		"%s#%s.mp4",
// 		strings.TrimSuffix(vf, filepath.Ext(vf)),
// 		suffix,
// 	)
// 	if err := utils.RunCmd(
// 		app.Config.Transcoder.Timeout,
// 		"ffmpeg", "-y", "-i", vf, "-s", size,
// 		"-c:v", "libx264", "-c:a", "aac",
// 		"-crf", "18", "-strict", "-2", "-loglevel", "quiet",
// 		"-metadata", fmt.Sprintf("title=%s", title),
// 		"-metadata", fmt.Sprintf("comment=%s", description),
// 		sf,
// 	); err != nil {
// 		err := fmt.Errorf("error transcoding video: %w", err)
// 		log.Error(err)
// 		http.Error(w, err.Error(), http.StatusInternalServerError)
// 		return
// 	}
// }
