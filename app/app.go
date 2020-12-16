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
	"sort"
	"strings"
	"time"

	rice "github.com/GeertJohan/go.rice"
	"github.com/dgrijalva/jwt-go"
	"github.com/dustin/go-humanize"
	"github.com/fsnotify/fsnotify"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/prologic/tube/importers"
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

	importTemplate := template.New("import").Funcs(templateFuncs)
	template.Must(importTemplate.Parse(box.MustString("import.html")))
	template.Must(importTemplate.Parse(box.MustString("base.html")))
	app.Templates.Add("import", importTemplate)

	// Setup Router
	router := mux.NewRouter().StrictSlash(true)
	router.HandleFunc("/", app.indexHandler).Methods("GET", "OPTIONS")
	router.HandleFunc("/upload", app.uploadHandler).Methods("GET", "OPTIONS", "POST")
	router.HandleFunc("/import", app.importHandler).Methods("GET", "OPTIONS", "POST")
	router.HandleFunc("/v/{id}.mp4", app.videoHandler).Methods("GET")
	router.HandleFunc("/v/{prefix}/{id}.mp4", app.videoHandler).Methods("GET")
	router.HandleFunc("/t/{id}", app.thumbHandler).Methods("GET")
	router.HandleFunc("/t/{prefix}/{id}", app.thumbHandler).Methods("GET")
	router.HandleFunc("/v/{id}", app.pageHandler).Methods("GET")
	router.HandleFunc("/v/{prefix}/{id}", app.pageHandler).Methods("GET")
	router.HandleFunc("/feed.xml", app.rssHandler).Methods("GET")
	
	router.HandleFunc("/auth/signup", app.apiCreateUserHandler).Methods("POST", "OPTIONS")
	router.HandleFunc("/auth/login", app.loginHandler).Methods("POST", "OPTIONS")
	
	api := router.PathPrefix("/api").Subrouter()
	api.Use(app.JwtVerify)
	router.HandleFunc("/upload", app.loginHandler).Methods("POST", "OPTIONS")

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

// HTTP handler for /upload
func (app *App) uploadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		ctx := map[string]interface{}{
			"MAX_UPLOAD_SIZE": app.Config.Server.MaxUploadSize,
		}
		app.render("upload", w, ctx)
	} else if r.Method == "POST" {
		r.ParseMultipartForm(app.Config.Server.MaxUploadSize)

		file, handler, err := r.FormFile("video_file")
		if err != nil {
			err := fmt.Errorf("error processing form: %w", err)
			log.Error(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer file.Close()

		title := r.FormValue("video_title")
		description := r.FormValue("video_description")

		// TODO: Make collection user selectable from drop-down in Form
		// XXX: Assume we can put uploaded videos into the first collection (sorted) we find
		keys := make([]string, 0, len(app.Library.Paths))
		for k := range app.Library.Paths {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		collection := keys[0]

		uf, err := ioutil.TempFile(
			app.Config.Server.UploadPath,
			fmt.Sprintf("tube-upload-*%s", filepath.Ext(handler.Filename)),
		)
		if err != nil {
			err := fmt.Errorf("error creating temporary file for uploading: %w", err)
			log.Error(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer os.Remove(uf.Name())

		_, err = io.Copy(uf, file)
		if err != nil {
			err := fmt.Errorf("error writing file: %w", err)
			log.Error(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		tf, err := ioutil.TempFile(
			app.Config.Server.UploadPath,
			fmt.Sprintf("tube-transcode-*.mp4"),
		)
		if err != nil {
			err := fmt.Errorf("error creating temporary file for transcoding: %w", err)
			log.Error(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		vf := filepath.Join(
			app.Library.Paths[collection].Path,
			fmt.Sprintf("%s.mp4", shortuuid.New()),
		)
		thumbFn1 := fmt.Sprintf("%s.jpg", strings.TrimSuffix(tf.Name(), filepath.Ext(tf.Name())))
		thumbFn2 := fmt.Sprintf("%s.jpg", strings.TrimSuffix(vf, filepath.Ext(vf)))

		// TODO: Use a proper Job Queue and make this async
		if err := utils.RunCmd(
			app.Config.Transcoder.Timeout,
			"ffmpeg",
			"-y",
			"-i", uf.Name(),
			"-vcodec", "h264",
			"-acodec", "aac",
			"-strict", "-2",
			"-loglevel", "quiet",
			"-metadata", fmt.Sprintf("title=%s", title),
			"-metadata", fmt.Sprintf("comment=%s", description),
			tf.Name(),
		); err != nil {
			err := fmt.Errorf("error transcoding video: %w", err)
			log.Error(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if err := utils.RunCmd(
			app.Config.Thumbnailer.Timeout,
			"mt",
			"-b",
			"-s",
			"-n", "1",
			tf.Name(),
		); err != nil {
			err := fmt.Errorf("error generating thumbnail: %w", err)
			log.Error(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if err := os.Rename(thumbFn1, thumbFn2); err != nil {
			err := fmt.Errorf("error renaming generated thumbnail: %w", err)
			log.Error(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if err := os.Rename(tf.Name(), vf); err != nil {
			err := fmt.Errorf("error renaming transcoded video: %w", err)
			log.Error(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// TODO: Make this a background job
		// Resize for lower quality options
		for size, suffix := range app.Config.Transcoder.Sizes {
			log.
				WithField("size", size).
				WithField("vf", filepath.Base(vf)).
				Info("resizing video for lower quality playback")
			sf := fmt.Sprintf(
				"%s#%s.mp4",
				strings.TrimSuffix(vf, filepath.Ext(vf)),
				suffix,
			)

			if err := utils.RunCmd(
				app.Config.Transcoder.Timeout,
				"ffmpeg",
				"-y",
				"-i", vf,
				"-s", size,
				"-c:v", "libx264",
				"-c:a", "aac",
				"-crf", "18",
				"-strict", "-2",
				"-loglevel", "quiet",
				"-metadata", fmt.Sprintf("title=%s", title),
				"-metadata", fmt.Sprintf("comment=%s", description),
				sf,
			); err != nil {
				err := fmt.Errorf("error transcoding video: %w", err)
				log.Error(err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}

		fmt.Fprintf(w, "Video successfully uploaded!")
	} else {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	}
}

// HTTP handler for /import
func (app *App) importHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		ctx := &struct{}{}
		app.render("import", w, ctx)
	} else if r.Method == "POST" {
		r.ParseMultipartForm(1024)

		url := r.FormValue("url")
		if url == "" {
			err := fmt.Errorf("error, no url supplied")
			log.Error(err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// TODO: Make collection user selectable from drop-down in Form
		// XXX: Assume we can put uploaded videos into the first collection (sorted) we find
		keys := make([]string, 0, len(app.Library.Paths))
		for k := range app.Library.Paths {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		collection := keys[0]

		videoImporter, err := importers.NewImporter(url)
		if err != nil {
			err := fmt.Errorf("error creating video importer for %s: %w", url, err)
			log.Error(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		videoInfo, err := videoImporter.GetVideoInfo(url)
		if err != nil {
			err := fmt.Errorf("error retriving video info for %s: %w", url, err)
			log.Error(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		uf, err := ioutil.TempFile(
			app.Config.Server.UploadPath,
			fmt.Sprintf("tube-import-*.mp4"),
		)
		if err != nil {
			err := fmt.Errorf("error creating temporary file for importing: %w", err)
			log.Error(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer os.Remove(uf.Name())

		log.WithField("video_url", videoInfo.VideoURL).Info("requesting video size")

		res, err := http.Head(videoInfo.VideoURL)
		if err != nil {
			err := fmt.Errorf("error getting size of video %w", err)
			log.Error(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		contentLength := utils.SafeParseInt64(res.Header.Get("Content-Length"), -1)
		if contentLength == -1 {
			err := fmt.Errorf("error calculating size of video")
			log.WithField("contentLength", contentLength).Error(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if contentLength > app.Config.Server.MaxUploadSize {
			err := fmt.Errorf(
				"imported video would exceed maximum upload size of %s",
				humanize.Bytes(uint64(app.Config.Server.MaxUploadSize)),
			)
			log.
				WithField("contentLength", contentLength).
				WithField("max_upload_size", app.Config.Server.MaxUploadSize).
				Error(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		log.WithField("contentLength", contentLength).Info("downloading video")

		if err := utils.Download(videoInfo.VideoURL, uf.Name()); err != nil {
			err := fmt.Errorf("error downloading video %s: %w", url, err)
			log.Error(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		tf, err := ioutil.TempFile(
			app.Config.Server.UploadPath,
			fmt.Sprintf("tube-transcode-*.mp4"),
		)
		if err != nil {
			err := fmt.Errorf("error creating temporary file for transcoding: %w", err)
			log.Error(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		vf := filepath.Join(
			app.Library.Paths[collection].Path,
			fmt.Sprintf("%s.mp4", shortuuid.New()),
		)
		thumbFn1 := fmt.Sprintf("%s.jpg", strings.TrimSuffix(tf.Name(), filepath.Ext(tf.Name())))
		thumbFn2 := fmt.Sprintf("%s.jpg", strings.TrimSuffix(vf, filepath.Ext(vf)))

		if err := utils.Download(videoInfo.ThumbnailURL, thumbFn1); err != nil {
			err := fmt.Errorf("error downloading thumbnail: %w", err)
			log.Error(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// TODO: Use a proper Job Queue and make this async
		if err := utils.RunCmd(
			app.Config.Transcoder.Timeout,
			"ffmpeg",
			"-y",
			"-i", uf.Name(),
			"-vcodec", "h264",
			"-acodec", "aac",
			"-strict", "-2",
			"-loglevel", "quiet",
			"-metadata", fmt.Sprintf("title=%s", videoInfo.Title),
			"-metadata", fmt.Sprintf("comment=%s", videoInfo.Description),
			tf.Name(),
		); err != nil {
			err := fmt.Errorf("error transcoding video: %w", err)
			log.Error(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if err := os.Rename(thumbFn1, thumbFn2); err != nil {
			err := fmt.Errorf("error renaming generated thumbnail: %w", err)
			log.Error(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if err := os.Rename(tf.Name(), vf); err != nil {
			err := fmt.Errorf("error renaming transcoded video: %w", err)
			log.Error(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// TODO: Make this a background job
		// Resize for lower quality options
		for size, suffix := range app.Config.Transcoder.Sizes {
			log.
				WithField("size", size).
				WithField("vf", filepath.Base(vf)).
				Info("resizing video for lower quality playback")
			sf := fmt.Sprintf(
				"%s#%s.mp4",
				strings.TrimSuffix(vf, filepath.Ext(vf)),
				suffix,
			)

			if err := utils.RunCmd(
				app.Config.Transcoder.Timeout,
				"ffmpeg",
				"-y",
				"-i", vf,
				"-s", size,
				"-c:v", "libx264",
				"-c:a", "aac",
				"-crf", "18",
				"-strict", "-2",
				"-loglevel", "quiet",
				"-metadata", fmt.Sprintf("title=%s", videoInfo.Title),
				"-metadata", fmt.Sprintf("comment=%s", videoInfo.Description),
				sf,
			); err != nil {
				err := fmt.Errorf("error transcoding video: %w", err)
				log.Error(err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}

		fmt.Fprintf(w, "Video successfully imported!")
	} else {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	}
}

// HTTP handler for /v/id
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

// HTTP handler for /v/id.mp4
func (app *App) videoHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	prefix, ok := vars["prefix"]
	if ok {
		id = path.Join(prefix, id)
	}

	log.Printf("/v/%s", id)

	m, ok := app.Library.Videos[id]
	if !ok {
		return
	}

	var videoPath string

	quality := strings.ToLower(r.URL.Query().Get("quality"))
	switch quality {
	case "720p", "480p", "360p", "240p":
		videoPath = fmt.Sprintf(
			"%s#%s.mp4",
			strings.TrimSuffix(m.Path, filepath.Ext(m.Path)),
			quality,
		)
		if !utils.FileExists(videoPath) {
			log.
				WithField("quality", quality).
				WithField("videoPath", videoPath).
				Warn("video with specified quality does not exist (defaulting to default quality)")
			videoPath = m.Path
		}
	case "":
		videoPath = m.Path
	default:
		log.WithField("quality", quality).Warn("invalid quality")
		videoPath = m.Path
	}

	if err := app.Store.Migrate(prefix, id); err != nil {
		err := fmt.Errorf("error migrating store data: %w", err)
		log.Warn(err)
	}

	if err := app.Store.IncViews(id); err != nil {
		err := fmt.Errorf("error updating view for %s: %w", id, err)
		log.Warn(err)
	}

	title := m.Title
	disposition := "attachment; filename=\"" + title + ".mp4\""
	w.Header().Set("Content-Disposition", disposition)
	w.Header().Set("Content-Type", "video/mp4")
	http.ServeFile(w, r, videoPath)
}

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
		app.panic(fmt.Errorf("Password encryption failed! %w", err), w)
		return
	}

	user.Password = string(pass)

	res := app.DataBase.Create(&user)
	if res.Error != nil {
		app.panic(fmt.Errorf("Failed to create user! %w", res.Error), w)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}


func (app *App) panic(e error, w http.ResponseWriter) {
	log.Error(e)
	w.WriteHeader(http.StatusBadRequest)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(models.ErrResponse{Error: e.Error()})
}

// HTTP handler for /auth/login
func (app *App) loginHandler(w http.ResponseWriter, r *http.Request) {
	user := &models.User{}
	err := json.NewDecoder(r.Body).Decode(user) 
	if err != nil {
		app.panic(err, w);
		return;
	}
	resp, err := app.findUser(user.Name, user.Password)
	if err != nil {
		app.panic(err, w);
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

func (app *App) JwtVerify(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get("x-access-token")
		header = strings.TrimSpace(header)

		if header == "" {
			next.ServeHTTP(w, r)
			return
		}

		tk := &models.CustomClaims{}
		_, err := jwt.ParseWithClaims(header, tk, func(token *jwt.Token) (interface{}, error) {
			return []byte("secret"), nil
		})

		if err != nil {
			next.ServeHTTP(w, r)
			return
		}

		ctx := context.WithValue(r.Context(), "user", tk)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (app *App) apiUploadVideoHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseMultipartForm(app.Config.Server.MaxUploadSize)

	file, handler, err := r.FormFile("video")
	if err != nil {
		err := fmt.Errorf("error processing form: %w", err)
		log.Error(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer file.Close()

	title := r.FormValue("title")
	description := r.FormValue("description")

	// TODO: Make collection user selectable from drop-down in Form
	// XXX: Assume we can put uploaded videos into the first collection (sorted) we find
	keys := make([]string, 0, len(app.Library.Paths))
	for k := range app.Library.Paths {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	collection := keys[0]

	uf, err := ioutil.TempFile(
		app.Config.Server.UploadPath,
		fmt.Sprintf("tube-upload-*%s", filepath.Ext(handler.Filename)),
	)
	if err != nil {
		app.panic(fmt.Errorf("error creating temporary file for uploading: %w", err), w);
		return
	}
	defer os.Remove(uf.Name())

	_, err = io.Copy(uf, file)
	if err != nil {
		app.panic(fmt.Errorf("error writing file: %w", err), w)
		return
	}

	tf, err := ioutil.TempFile(
		app.Config.Server.UploadPath,
		fmt.Sprintf("tube-transcode-*.mp4"),
	)
	if err != nil {
		app.panic(fmt.Errorf("error creating temporary file for transcoding: %w", err), w)
		return
	}

	vf := filepath.Join(
		app.Library.Paths[collection].Path,
		fmt.Sprintf("%s.mp4", shortuuid.New()),
	)
	thumbFn1 := fmt.Sprintf("%s.jpg", strings.TrimSuffix(tf.Name(), filepath.Ext(tf.Name())))
	thumbFn2 := fmt.Sprintf("%s.jpg", strings.TrimSuffix(vf, filepath.Ext(vf)))

	// TODO: Use a proper Job Queue and make this async
	if err := utils.RunCmd(
		app.Config.Transcoder.Timeout,
		"ffmpeg", "-y", "-i", uf.Name(),
		"-vcodec", "h264", "-acodec", "aac",
		"-strict", "-2", "-loglevel", "quiet",
		"-metadata", fmt.Sprintf("title=%s", title),
		"-metadata", fmt.Sprintf("comment=%s", description),
		tf.Name(),
	); err != nil {
		err := fmt.Errorf("error transcoding video: %w", err)
		log.Error(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = utils.RunCmd(app.Config.Thumbnailer.Timeout, "mt", "-b", "-s", "-n", "1", tf.Name()); 
	if err != nil {
		err := fmt.Errorf("error generating thumbnail: %w", err)
		log.Error(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := os.Rename(thumbFn1, thumbFn2); err != nil {
		err := fmt.Errorf("error renaming generated thumbnail: %w", err)
		log.Error(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := os.Rename(tf.Name(), vf); err != nil {
		err := fmt.Errorf("error renaming transcoded video: %w", err)
		log.Error(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// TODO: Make this a background job
	// Resize for lower quality options
	for size, suffix := range app.Config.Transcoder.Sizes {
		log.
			WithField("size", size).
			WithField("vf", filepath.Base(vf)).
			Info("resizing video for lower quality playback")
		sf := fmt.Sprintf(
			"%s#%s.mp4",
			strings.TrimSuffix(vf, filepath.Ext(vf)),
			suffix,
		)

		if err := utils.RunCmd(
			app.Config.Transcoder.Timeout,
			"ffmpeg",
			"-y",
			"-i", vf,
			"-s", size,
			"-c:v", "libx264",
			"-c:a", "aac",
			"-crf", "18",
			"-strict", "-2",
			"-loglevel", "quiet",
			"-metadata", fmt.Sprintf("title=%s", title),
			"-metadata", fmt.Sprintf("comment=%s", description),
			sf,
		); err != nil {
			err := fmt.Errorf("error transcoding video: %w", err)
			log.Error(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	fmt.Fprintf(w, "Video successfully uploaded!")

}
