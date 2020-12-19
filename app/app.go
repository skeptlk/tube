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
	"os"
	"path"
	"path/filepath"
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
	app := &App {
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
	// router.HandleFunc("/upload", app.uploadHandler).Methods("GET", "OPTIONS", "POST")
	router.HandleFunc("/v/{id}.mp4", app.getVideoHandler).Methods("GET")
	router.HandleFunc("/v/{id}", app.getVideoInfoHandler).Methods("GET", "OPTIONS")
	router.HandleFunc("/t/{id}", app.thumbHandler).Methods("GET")
	router.HandleFunc("/t/{prefix}/{id}", app.thumbHandler).Methods("GET")
	router.HandleFunc("/feed.xml", app.rssHandler).Methods("GET")
	
	router.HandleFunc("/auth/signup", app.apiCreateUserHandler).Methods("POST", "OPTIONS")
	router.HandleFunc("/auth/login", app.loginHandler).Methods("POST", "OPTIONS")
	
	api := router.PathPrefix("/api").Subrouter()
	api.Use(app.JwtVerify)
	api.HandleFunc("/video", app.apiUploadVideoHandler).Methods("POST", "OPTIONS")

	// Static file handler
	fsHandler := http.StripPrefix(
		"/static",
		http.FileServer(rice.MustFindBox("../static").HTTPBox()),
	)
	router.PathPrefix("/static/").Handler(fsHandler).Methods("GET")

	cors := handlers.CORS(
		handlers.AllowedHeaders([]string{
			"X-Requested-With",
			"Content-Type",
			"Authorization",
		}),
		handlers.AllowedMethods([]string{
			"GET",
			"POST",
			"PUT",
			"HEAD",
			"OPTIONS",
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
	
	app.DataBase, err = ConnectDB();
	if err != nil {
		return err
	}

	for _, pc := range app.Config.Library {
		p := &media.Path {
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


// HTTP handler for /v/id
/*
func (app *App) pageHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	prefix, ok := vars["prefix"]
	if ok {
		id = path.Join(prefix, id)
	}
	log.Printf("/v/%s", id)
	playing, ok := app.Library.Videos[id]
	if !ok {
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
		app.render("upload", w, ctx)
		return
	}

	views, err := app.Store.GetViews(id)
	if err != nil {
		err := fmt.Errorf("error retrieving views for %s: %w", id, err)
		log.Warn(err)
	}

	playing.Views = views

	playlist := app.Library.Playlist()

	// TODO: Optimize this? Bitcask has no concept of MultiGet / MGET
	for _, video := range playlist {
		views, err := app.Store.GetViews(video.ID)
		if err != nil {
			err := fmt.Errorf("error retrieving views for %s: %w", video.ID, err)
			log.Warn(err)
		}
		video.Views = views
	}

	sort := strings.ToLower(r.URL.Query().Get("sort"))
	switch sort {
	case "views":
		media.By(media.SortByViews).Sort(playlist)
	case "", "timestamp":
		media.By(media.SortByTimestamp).Sort(playlist)
	default:
		// By default the playlist is sorted by Timestamp
		log.Warnf("invalid sort critiera: %s", sort)
	}

	quality := strings.ToLower(r.URL.Query().Get("quality"))
	switch quality {
	case "", "720p", "480p", "360p", "240p":
	default:
		log.WithField("quality", quality).Warn("invalid quality")
		quality = ""
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	ctx := &struct {
		Sort     string
		Quality  string
		Playing  *media.Video
		Playlist media.Playlist
	}{
		Sort:     sort,
		Quality:  quality,
		Playing:  playing,
		Playlist: playlist,
	}
	app.render("index", w, ctx)
}
*/

// HTTP handler for /t/id
func (app *App) thumbHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	prefix, ok := vars["prefix"]
	if ok {
		id = path.Join(prefix, id)
	}
	log.Printf("/t/%s", id)
	m, ok := app.Library.Videos[id]
	if !ok {
		return
	}
	w.Header().Set("Cache-Control", "public, max-age=7776000")
	if m.ThumbType == "" {
		w.Header().Set("Content-Type", "image/jpeg")
		w.Write(rice.MustFindBox("../static").MustBytes("defaulticon.jpg"))
	} else {
		w.Header().Set("Content-Type", m.ThumbType)
		w.Write(m.Thumb)
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
	user := &models.User {}
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
	user := &models.User{}
	err := json.NewDecoder(r.Body).Decode(user) 
	if err != nil {
		http.Error(w, "Login failed", http.StatusBadRequest)
		log.Error(err)
		return;
	}
	resp, err := app.findUser(user.Name, user.Password)
	if err != nil {
		http.Error(w, "Login failed", http.StatusBadRequest)
		log.Error(err)
		return
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

	// DO I NEED TO INCLUDE PASSWORD HERE? I THINK SO...
	tk := &models.CustomClaims{
		UserID: user.ID, 
		Name: user.Name, 
		Email: user.Email,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: expiresAt,
		},
	}

	token := jwt.NewWithClaims(jwt.GetSigningMethod("HS256"), tk)
	tokenString, error := token.SignedString([]byte("secret"))
	if error != nil {
		fmt.Println(error)
	}

	var resp = map[string] interface{} {"status": false, "message": "Logged in successfully!"}
	resp["token"] = tokenString //Store the token in the response
	resp["user"] = user
	return resp, nil
}

// MIDDLEWARE FOR AUTHENTICATION
func (app *App) JwtVerify(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get("Authorization")
		header = strings.TrimSpace(header)

		if header == "" {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("Token is not provided"))
			return
		}

		tk := &models.CustomClaims{}
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

// HTTP handler for /api/upload
func (app *App) apiUploadVideoHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseMultipartForm(app.Config.Server.MaxUploadSize)

	file, handler, err := r.FormFile("video")
	if err != nil {
		log.Error(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer file.Close()

	// CREATE VIDEO OBJECT
	vid := &models.Video {}

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
	defer os.Remove(tempCopy.Name())

	_, err = io.Copy(tempCopy, file)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		log.Error(err, w)
		return
	}

	transcodeFile, err := ioutil.TempFile(
		app.Config.Server.UploadPath,
		fmt.Sprintf("tube-transcode-*.mp4"),
	)
	log.Info(fmt.Sprintf("Transcode file name: %s", transcodeFile.Name()))
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		log.Error(err, w)
		return
	}

	// CREATE THUMBNAIL
	uniqueName := fmt.Sprintf("%s.mp4", shortuuid.New())
	destVid := filepath.Join(app.Config.Server.UploadPath, uniqueName)
	tempThumb := fmt.Sprintf("%s.jpg", strings.TrimSuffix(transcodeFile.Name(), ".mp4"))
	destThumb := fmt.Sprintf("%s.jpg", strings.TrimSuffix(destVid, filepath.Ext(destVid)))

	vid.URL = destVid;
	vid.ThumbnailURL = destThumb;

	if err := utils.RunCmd(
		app.Config.Transcoder.Timeout,
		"ffmpeg", "-y", "-i", tempCopy.Name(),
		"-vcodec", "h264", "-acodec", "aac",
		"-strict", "-2", "-loglevel", "quiet",
		"-metadata", fmt.Sprintf("title=%s", vid.Title),
		"-metadata", fmt.Sprintf("comment=%s", vid.Description),
		transcodeFile.Name(),
	); err != nil {
		log.Error(fmt.Errorf("Error transcoding video: %w", err))
		http.Error(w, "Can't process this file", http.StatusInternalServerError)
		return
	}

	err = utils.RunCmd(app.Config.Thumbnailer.Timeout, 
		"mt", "-b", "-s", "-n", "1", 
		transcodeFile.Name()); 
	if err != nil {
		log.Error(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := os.Rename(tempThumb, destThumb); err != nil {
		log.Error(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := os.Rename(transcodeFile.Name(), destVid); err != nil {
		log.Error(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	res := app.DataBase.Create(&vid)
	if res.Error != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		log.Error(res.Error)
		return
	}

	fmt.Fprintf(w, "Video successfully uploaded!")
}

// HTTP handler for /v/id.mp4
func (app *App) getVideoHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	log.Printf("/v/%s.mp4", id)

	video := &models.Video {}
	app.DataBase.First(video, id);

	if (video.ID > 0) {
		_, filename := path.Split(video.URL)
		disposition := `attachment; filename="` + filename + `"`
		w.Header().Set("Content-Disposition", disposition)
		w.Header().Set("Content-Type", "video/mp4")
		http.ServeFile(w, r, video.URL)
		
		defer app.incrementViews(video);
	} else {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("Video not found"))
	}
}

func (app *App) incrementViews(vid *models.Video) {
	vid.Views++
	app.DataBase.Save(&vid)
}

// HTTP handler for /v/id
func (app *App) getVideoInfoHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	log.Printf("/v/%s", id)

	video := &models.Video {}
	app.DataBase.First(video, id);

	if (video.ID > 0) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(video)
	} else {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("Video not found"))
	}
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

