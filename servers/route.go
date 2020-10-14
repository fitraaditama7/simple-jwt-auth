package servers

import (
	"github.com/gin-gonic/contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/simple-jwt-auth/api"
	"github.com/simple-jwt-auth/middleware"
	"github.com/simple-jwt-auth/models"
	"golang.org/x/oauth2"
	githuboauth "golang.org/x/oauth2/github"
	"golang.org/x/oauth2/google"
	"log"
)

func (server *Server) InitializeRoutes() {
	casbinService := api.NewCasbinService(server.Enforcer)
	userRepos := models.ProvideUserRepository(server.DB)
	jwtApi := api.CreateJwtApi(&userRepos)
	token, err := middleware.RandToken(64)
	if err != nil {
		log.Fatal("unable to generate random token: ", err)
	}
	store := sessions.NewCookieStore([]byte(token))
	store.Options(sessions.Options{
		Path:   "/",
		MaxAge: 86400 * 7,
	})
	googleConf := oauth2.Config{
		ClientID:     server.enviroment.GoogleConf.ClientID,
		ClientSecret: server.enviroment.GoogleConf.ClientSecret,
		Endpoint:     google.Endpoint,
		RedirectURL:  server.enviroment.GoogleConf.RedirectUrl,
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.email", // You have to select your own scope from here -> https://developers.google.com/identity/protocols/googlescopes#google_sign-in
		},
	}

	googleApi := api.GoogleAPI{
		Config:   &googleConf,
		UserRepo: &userRepos,
	}

	githubConf := &oauth2.Config{
		ClientID:     "Iv1.f2915f579568fa22",
		ClientSecret: "93c5b51a97cefef9f46d18f459b9f34aab838a12",
		Scopes:       []string{"user:email", "repo"},
		Endpoint:     githuboauth.Endpoint,
	}

	githubApi := api.GithubAPI{
		Config:   githubConf,
		UserRepo: &userRepos,
	}
	// jwt api
	server.Router.POST("/login/token", jwtApi.JwtLogin)
	server.Router.Use(gin.Logger())
	server.Router.Use(gin.Recovery())
	server.Router.Static("/css", "./static/css")
	server.Router.Static("/img", "./static/img")
	server.Router.LoadHTMLGlob("./static/templates/*")

	// jwt
	jwt := server.Router.Group("/api")
	jwt.Use(middleware.TokenAuthMiddleware())
	{
		jwt.POST("/auth/policy", middleware.AuthorizeJwtToken("/auth/policy", "POST", server.Enforcer), casbinService.CreatePolicy)
		jwt.GET("/auth/policy", middleware.AuthorizeJwtToken("/auth/policy", "GET", server.Enforcer), casbinService.ListPolicy)
		jwt.POST("/auth/grouppolicy", middleware.AuthorizeJwtToken("/auth/grouppolicy", "POST", server.Enforcer), casbinService.CreateGroupPolicy)
		jwt.GET("/auth/grouppolicy", middleware.AuthorizeJwtToken("/auth/grouppolicy", "GET", server.Enforcer), casbinService.ListGroupPolicies)
		//jwt.POST("/todo", middleware.AuthorizeJwtToken("resource", "write", server.Enforcer), api.CreateTodo)
		//jwt.GET("/todo", middleware.AuthorizeJwtToken("resource", "read", server.Enforcer), api.GetTodo)
		jwt.POST("/logout", jwtApi.JwtLogout)
		jwt.POST("/refresh", jwtApi.JwtRefresh)
	}

	// googleid api
	googleOauth := server.Router.Group("/oauth/google")
	googleOauth.Use(sessions.Sessions("goquestsession", store))
	googleOauth.GET("/", googleApi.IndexHandler)

	googleOauth.GET("/login", googleApi.LoginHandler)
	googleOauth.GET("/auth", googleApi.AuthHandler)
	googleOauth.Use(middleware.AuthorizeOpenIdRequest())
	{
		googleOauth.GET("/field", googleApi.FieldHandler)
		googleOauth.GET("/test", googleApi.TestHandler)
	}

	//github api
	githubOauth := server.Router.Group("/oauth/github")
	githubOauth.Use(sessions.Sessions("goquestsession", store))
	githubOauth.GET("/", githubApi.IndexHandler)

	githubOauth.GET("/login", githubApi.LoginHandler)
	githubOauth.GET("/auth", githubApi.AuthHandler)
	githubOauth.Use(middleware.AuthorizeOpenIdRequest())
	{
		githubOauth.GET("/field", githubApi.FieldHandler)
		githubOauth.GET("/test", githubApi.TestHandler)
	}
}
