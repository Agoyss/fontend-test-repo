package routes

import (
	"database/sql"
	"net/http"

	"frontend-test/handlers"

	"github.com/gin-gonic/gin"
)

// RegisterRoutes wires article API endpoints and serves the dashboard.
func RegisterRoutes(r *gin.Engine, db *sql.DB) {
	// 1. Create
	r.POST("/article/", func(c *gin.Context) { handlers.CreateArticle(c, db) })
	r.POST("/article", func(c *gin.Context) { handlers.CreateArticle(c, db) })

	// 2. List articles (publish + draft)  GET /article/<limit>/<offset>
	r.GET("/article/:a/:b", func(c *gin.Context) { handlers.ListArticles(c, db) })
	//    List by status (paginated)       GET /article/status/<status>/<limit>/<offset>
	r.GET("/article/status/:status/:limit/:offset", func(c *gin.Context) { handlers.ListArticlesByStatus(c, db) })
	//    Get one                         GET /article/<id>
	r.GET("/article/:a", func(c *gin.Context) { handlers.GetArticleByID(c, db) })

	// 3. Update  PUT/PATCH/POST /article/<id>
	r.PUT("/article/:id", func(c *gin.Context) { handlers.UpdateArticle(c, db) })
	r.PATCH("/article/:id", func(c *gin.Context) { handlers.UpdateArticle(c, db) })
	r.POST("/article/:id", func(c *gin.Context) { handlers.UpdateArticle(c, db) })

	//    Move to trash (PDF req 1c)
	r.POST("/article/:id/thrash", func(c *gin.Context) { handlers.ThrashArticle(c, db) })

	// 4. Delete  DELETE /article/<id>
	r.DELETE("/article/:id", func(c *gin.Context) { handlers.DeleteArticle(c, db) })

	// 5. Trash tab list  GET /trash/<limit>/<offset>
	r.GET("/trash/:a/:b", func(c *gin.Context) { handlers.ListTrash(c, db) })
	//    Restore from trash (move back to articles) / permanent delete
	r.POST("/trash/:id/restore", func(c *gin.Context) { handlers.RestoreTrash(c, db) })
	r.DELETE("/trash/:id", func(c *gin.Context) { handlers.DeleteTrash(c, db) })

	// 6. Dashboard (served by this same Go server, reuses DB connection)
	r.Static("/static", "./public")
	r.LoadHTMLFiles("./public/index.html")
	r.GET("/", func(c *gin.Context) { c.HTML(http.StatusOK, "index.html", nil) })
}
