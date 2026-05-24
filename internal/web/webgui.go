package web

import (
	"html/template"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/aceberg/ExerciseDiary/internal/auth"
	"github.com/aceberg/ExerciseDiary/internal/check"
	"github.com/aceberg/ExerciseDiary/internal/conf"
	"github.com/aceberg/ExerciseDiary/internal/db"
)

// Gui - start web server
func Gui(dirPath, nodePath string) {

	confPath := dirPath + "/config.yaml"
	check.Path(confPath)

	appConfig, authConf = conf.Get(confPath)

	appConfig.DirPath = dirPath
	appConfig.DBPath = dirPath + "/sqlite.db"
	check.Path(appConfig.DBPath)
	appConfig.ConfPath = confPath
	appConfig.NodePath = nodePath
	appConfig.Icon = icon

	log.Println("INFO: starting web gui with config", appConfig.ConfPath)

	db.Create(appConfig.DBPath)
	db.EnsureSchema(appConfig.DBPath)
	db.MigrateMultiUser(appConfig.DBPath)
	recomputeAllPRTScores(appConfig.DBPath)

	address := appConfig.Host + ":" + appConfig.Port

	log.Println("=================================== ")
	log.Printf("Web GUI at http://%s", address)
	log.Println("=================================== ")

	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()

	funcMap := template.FuncMap{
		"ytEmbed":       YouTubeEmbed,
		"prtMMSS":       prtFormatSeconds,
		"prtEventLabel": prtEventLabel,
		"prtEventBadge": prtEventBadge,
		"prtCatClass":   prtCatClass,
	}
	templ := template.Must(template.New("").Funcs(funcMap).ParseFS(templFS, "templates/*"))
	router.SetHTMLTemplate(templ) // templates

	router.StaticFS("/fs/", http.FS(pubFS)) // public

	router.GET("/login/", loginHandler)  // login.go
	router.POST("/login/", loginHandler) // login.go

	router.Use(userMiddleware()) // middleware_user.go

	router.GET("/", auth.Auth(&authConf), indexHandler)             // index.go
	router.GET("/config/", auth.Auth(&authConf), configHandler)     // config.go
	router.GET("/exercise/", auth.Auth(&authConf), exerciseHandler) // exercise.go
	router.GET("/stats/", auth.Auth(&authConf), statsHandler)       // stats.go
	router.GET("/weight/", auth.Auth(&authConf), weightHandler)     // weight.go
	router.GET("/users/", auth.Auth(&authConf), usersHandler)       // users.go
	router.GET("/prt/", auth.Auth(&authConf), prtListHandler)       // prt.go
	router.GET("/prt/new", auth.Auth(&authConf), prtFormHandler)    // prt.go
	router.GET("/prt/edit/:id", auth.Auth(&authConf), prtFormHandler) // prt.go

	router.POST("/config/", auth.Auth(&authConf), saveConfigHandler)     // config.go
	router.POST("/config/auth", auth.Auth(&authConf), saveConfigAuth)    // config.go
	router.POST("/exercise/", auth.Auth(&authConf), saveExerciseHandler) // exercise.go
	router.POST("/exdel/", auth.Auth(&authConf), deleteExerciseHandler)  // exercise.go
	router.POST("/set/", auth.Auth(&authConf), setHandler)               // set.go
	router.POST("/weight/", auth.Auth(&authConf), addWeightHandler)      // weight.go
	router.POST("/users/", auth.Auth(&authConf), saveUserHandler)        // users.go
	router.POST("/userdel/", auth.Auth(&authConf), deleteUserHandler)    // users.go
	router.POST("/prt/", auth.Auth(&authConf), prtSaveHandler)           // prt.go
	router.POST("/prtdel/", auth.Auth(&authConf), prtDeleteHandler)      // prt.go
	router.POST("/user/switch", switchUserHandler)                       // middleware_user.go

	err := router.Run(address)
	check.IfError(err)
}
