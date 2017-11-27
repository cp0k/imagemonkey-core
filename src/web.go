package main

	
import (
	"net/http"
	"html/template"
	"github.com/gin-gonic/gin"
	"fmt"
	"strings"
	"os"
	log "github.com/Sirupsen/logrus"
	"flag"
	"database/sql"
	"math"
)

var db *sql.DB

func main() {
	fmt.Printf("Starting Web Service...\n")

	log.SetLevel(log.DebugLevel)

	fmt.Printf("Setting environment variable for sentry\n")
	os.Setenv("SENTRY_DSN", WEB_SENTRY_DSN)

	releaseMode := flag.Bool("release", false, "Run in release mode")
	wordlistPath := flag.String("wordlist", "../wordlists/en/labels.json", "Path to labels map")
	donationsDir := flag.String("donations_dir", "../donations/", "Location of the uploaded and verified donations")
	apiBaseUrl := flag.String("api_base_url", "http://127.0.0.1:8081", "API Base URL")
	playgroundBaseUrl := flag.String("playground_base_url", "http://127.0.0.1:8081", "Playground Base URL")
	htmlDir := flag.String("html_dir", "../html/templates/", "Location of the html directory")

	webAppIdentifier := "edd77e5fb6fc0775a00d2499b59b75d"

	flag.Parse()
	if(*releaseMode){
		fmt.Printf("Starting gin in release mode!\n")
		gin.SetMode(gin.ReleaseMode)
	}

	funcMap := template.FuncMap{
	    //simple round function
	    //be careful: only works for POSITIVE float values
	    "round" : func(f float32, places int) (float64) {
		    shift := math.Pow(10, float64(places))
		    return math.Floor((float64(f) * shift) + .5) / shift;    
		},
	}

	log.Debug("[Main] Reading Label Map")
	labelMap, words, err := getLabelMap(*wordlistPath)
	if err != nil {
		fmt.Printf("[Main] Couldn't read label map...terminating!")
		log.Fatal(err)
	}

	//open database and make sure that we can ping it
	db, err = sql.Open("postgres", IMAGE_DB_CONNECTION_STRING)
	if err != nil {
		log.Fatal("[Main] Couldn't open database: ", err.Error())
	}

	err = db.Ping()
	if err != nil {
		log.Fatal("[Main] Couldn't ping database: ", err.Error())
	}

	tmpl := template.Must(template.New("main").Funcs(funcMap).ParseGlob(*htmlDir + "*"))


	router := gin.Default()
	router.SetHTMLTemplate(tmpl)
	router.Static("./js", "../js") //serve javascript files
	router.Static("./css", "../css") //serve css files
	router.Static("./img", "../img") //serve images
	router.Static("./api", "../html/static/api")
	router.Static("./donations", *donationsDir) //serve doncations
	router.Static("./blog", "../html/static/blog")
	router.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", gin.H{
			"title": "ImageMonkey",
			"activeMenuNr": 1,
			"numOfDonations": pick(getNumOfDonatedImages())[0],
		})
	})
	router.GET("/donate", func(c *gin.Context) {
		c.HTML(http.StatusOK, "donate.html", gin.H{
			"title": "Donate Image",
			"randomWord": words[random(0, len(words) - 1)],
			"activeMenuNr": 2,
			"apiBaseUrl": apiBaseUrl,
			"words": words,
			"appIdentifier": webAppIdentifier,
		})
	})

	router.GET("/label", func(c *gin.Context) {
		c.HTML(http.StatusOK, "label.html", gin.H{
			"title": "Add Labels",
			"image": pick(getImageToLabel())[0],
			"activeMenuNr": 3,
			"apiBaseUrl": apiBaseUrl,
			"labels": labelMap,
		})
	})


	router.GET("/annotate", func(c *gin.Context) {
		c.HTML(http.StatusOK, "annotate.html", gin.H{
			"title": "Annotate",
			"randomImage": getRandomUnannotatedImage(),
			"activeMenuNr": 4,
			"apiBaseUrl": apiBaseUrl,
			"appIdentifier": webAppIdentifier,
		})
	})

	router.GET("/verify", func(c *gin.Context) {
		params := c.Request.URL.Query()
		
		showHeader := true
		if temp, ok := params["show_header"]; ok {
			if temp[0] == "false" {
				showHeader = false
			}
		}

		showFooter := true
		if temp, ok := params["show_footer"]; ok {
			if temp[0] == "false" {
				showFooter = false
			}
		}

		onlyOnce := false
		if temp, ok := params["only_once"]; ok {
			if temp[0] == "true" {
				onlyOnce = true
			}
		}


		c.HTML(http.StatusOK, "validate.html", gin.H{
			"title": "Validate Label",
			"randomImage": getRandomImage(),
			"activeMenuNr": 5,
			"showHeader": showHeader,
			"showFooter": showFooter,
			"onlyOnce": onlyOnce,
			"apiBaseUrl": apiBaseUrl,
			"appIdentifier": webAppIdentifier,
		})
	})
	router.GET("/verify_annotation", func(c *gin.Context) {
		c.HTML(http.StatusOK, "validate_annotations.html", gin.H{
			"title": "Validate Annotations",
			"randomImage": pick(getRandomAnnotatedImage())[0],
			"activeMenuNr": 6,
			"apiBaseUrl": apiBaseUrl,
			"appIdentifier": webAppIdentifier,
		})
	})
	router.GET("/quiz", func(c *gin.Context) {
		c.HTML(http.StatusOK, "quiz.html", gin.H{
			"title": "Quiz",
			"randomQuiz": "",
			"randomAnnotatedImage": pick(getRandomAnnotationForRefinement())[0],
			"activeMenuNr": 7,
			"apiBaseUrl": apiBaseUrl,
		})
	})	
	router.GET("/explore", func(c *gin.Context) {
		c.HTML(http.StatusOK, "explore.html", gin.H{
			"title": "Explore Dataset",
			"words": words,
			"activeMenuNr": 8,
			"statistics": pick(explore(words))[0],
		})
	})
	router.GET("/export", func(c *gin.Context) {
		c.HTML(http.StatusOK, "export.html", gin.H{
			"title": "Export Dataset",
			"labels": pick(getAllImageLabels())[0],
			"activeMenuNr": 9,
		})
	})
	router.GET("/apps", func(c *gin.Context) {
		c.HTML(http.StatusOK, "mobile.html", gin.H{
			"title": "Mobile Apps & Extensions",
			"activeMenuNr": 10,
		})
	})
	router.GET("/playground", func(c *gin.Context) {
		c.HTML(http.StatusOK, "playground.html", gin.H{
			"title": "Playground",
			"activeMenuNr": 11,
			"playgroundPredictBaseUrl": playgroundBaseUrl,
		})
	})

	router.GET("/data", func(c *gin.Context) {
		tags := ""
		params := c.Request.URL.Query()
		if temp, ok := params["tags"]; ok {
			tags = temp[0]
			jsonData, err := export(strings.Split(tags, ","))
			if(err == nil){
				c.JSON(http.StatusOK, jsonData)
				return
			} else{
				c.JSON(http.StatusInternalServerError, gin.H{"Error": "Couldn't export data"})
				return
			}
		} else {
			c.JSON(422, gin.H{"error": "No tags specified"})
			return
		}
	})

	router.Run(":8080")
}
