package handlers

import (
	"database/sql"
	"net/http"
	"strconv"

	"frontend-test/models"

	"github.com/gin-gonic/gin"
)

// validate enforces only what the column types can't (status enum) and
// caps lengths to the column definitions (title VARCHAR(200), category VARCHAR(100)).
func validate(in models.ArticleInput) map[string]string {
	errs := map[string]string{}
	if len(in.Title) > 200 {
		errs["title"] = "title must not exceed 200 characters"
	}
	if len(in.Category) > 100 {
		errs["category"] = "category must not exceed 100 characters"
	}
	if in.Status != "publish" && in.Status != "draft" && in.Status != "thrash" {
		errs["status"] = "status must be publish, draft or thrash"
	}
	return errs
}

const articleCols = "id, title, content, category, status"

func scanArticle(s interface {
	Scan(...interface{}) error
}) (models.Article, error) {
	var a models.Article
	err := s.Scan(&a.ID, &a.Title, &a.Content, &a.Category, &a.Status)
	return a, err
}

func CreateArticle(c *gin.Context, db *sql.DB) {
	var in models.ArticleInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}
	if errs := validate(in); len(errs) > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"message": "validation failed", "errors": errs})
		return
	}
	res, err := db.Exec(
		"INSERT INTO articles (title, content, category, status) VALUES (?, ?, ?, ?)",
		in.Title, in.Content, in.Category, in.Status,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}
	id, _ := res.LastInsertId()
	a, _ := getArticle(db, int(id))
	c.JSON(http.StatusCreated, a)
}

// ListArticles handles GET /article/<limit>/<offset> — rows from the
// articles table (publish + draft). The frontend filters by tab status.
func ListArticles(c *gin.Context, db *sql.DB) {
	limit, err1 := strconv.Atoi(c.Param("a"))
	offset, err2 := strconv.Atoi(c.Param("b"))
	if err1 != nil || err2 != nil || limit < 0 || offset < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"message": "limit and offset must be non-negative integers"})
		return
	}
	rows, err := db.Query(
		"SELECT "+articleCols+" FROM articles ORDER BY id DESC LIMIT ? OFFSET ?",
		limit, offset,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}
	defer rows.Close()

	list := []models.Article{}
	for rows.Next() {
		a, err := scanArticle(rows)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
			return
		}
		list = append(list, a)
	}
	if list == nil {
		list = []models.Article{}
	}
	c.JSON(http.StatusOK, list)
}

// ListArticlesByStatus handles GET /article/status/:status/:limit/:offset —
// rows from the articles table filtered by a single status, server-side
// paginated. Used by the dashboard tabs (Published / Drafts).
func ListArticlesByStatus(c *gin.Context, db *sql.DB) {
	status := c.Param("status")
	if status != "publish" && status != "draft" && status != "thrash" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "status must be publish, draft or thrash"})
		return
	}
	limit, err1 := strconv.Atoi(c.Param("limit"))
	offset, err2 := strconv.Atoi(c.Param("offset"))
	if err1 != nil || err2 != nil || limit < 0 || offset < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"message": "limit and offset must be non-negative integers"})
		return
	}
	rows, err := db.Query(
		"SELECT "+articleCols+" FROM articles WHERE status = ? ORDER BY id DESC LIMIT ? OFFSET ?",
		status, limit, offset,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}
	defer rows.Close()

	list := []models.Article{}
	for rows.Next() {
		a, err := scanArticle(rows)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
			return
		}
		list = append(list, a)
	}
	if list == nil {
		list = []models.Article{}
	}
	c.JSON(http.StatusOK, list)
}

func getArticle(db *sql.DB, id int) (models.Article, bool) {
	var a models.Article
	err := db.QueryRow(
		"SELECT "+articleCols+" FROM articles WHERE id = ?", id,
	).Scan(&a.ID, &a.Title, &a.Content, &a.Category, &a.Status)
	if err == sql.ErrNoRows {
		return a, false
	}
	return a, err == nil
}

func GetArticleByID(c *gin.Context, db *sql.DB) {
	id, err := strconv.Atoi(c.Param("a"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "id must be an integer"})
		return
	}
	if a, ok := getArticle(db, id); ok {
		c.JSON(http.StatusOK, a)
	} else {
		c.JSON(http.StatusNotFound, gin.H{"message": "article not found"})
	}
}

func UpdateArticle(c *gin.Context, db *sql.DB) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "id must be an integer"})
		return
	}
	var in models.ArticleInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}
	if errs := validate(in); len(errs) > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"message": "validation failed", "errors": errs})
		return
	}
	res, err := db.Exec(
		"UPDATE articles SET title = ?, content = ?, category = ?, status = ? WHERE id = ?",
		in.Title, in.Content, in.Category, in.Status, id,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		c.JSON(http.StatusNotFound, gin.H{"message": "article not found"})
		return
	}
	a, _ := getArticle(db, id)
	c.JSON(http.StatusOK, a)
}

func DeleteArticle(c *gin.Context, db *sql.DB) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "id must be an integer"})
		return
	}
	res, err := db.Exec("DELETE FROM articles WHERE id = ?", id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		c.JSON(http.StatusNotFound, gin.H{"message": "article not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{})
}

// ThrashArticle moves an article from the articles table into the trash
// table (PDF req 1c: thrash icon -> article moves to Trashed tab).
func ThrashArticle(c *gin.Context, db *sql.DB) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "id must be an integer"})
		return
	}
	a, ok := getArticle(db, id)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"message": "article not found"})
		return
	}
	tx, err := db.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}
	if _, err := tx.Exec(
		"INSERT INTO trash (title, content, category, status) VALUES (?, ?, ?, 'thrash')",
		a.Title, a.Content, a.Category,
	); err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}
	if _, err := tx.Exec("DELETE FROM articles WHERE id = ?", id); err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}
	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "moved to trash"})
}

// ListTrash handles GET /trash/<limit>/<offset> — rows from the trash table.
func ListTrash(c *gin.Context, db *sql.DB) {
	limit, err1 := strconv.Atoi(c.Param("a"))
	offset, err2 := strconv.Atoi(c.Param("b"))
	if err1 != nil || err2 != nil || limit < 0 || offset < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"message": "limit and offset must be non-negative integers"})
		return
	}
	rows, err := db.Query(
		"SELECT "+articleCols+" FROM trash ORDER BY id DESC LIMIT ? OFFSET ?",
		limit, offset,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}
	defer rows.Close()

	list := []models.Article{}
	for rows.Next() {
		a, err := scanArticle(rows)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
			return
		}
		list = append(list, a)
	}
	if list == nil {
		list = []models.Article{}
	}
	c.JSON(http.StatusOK, list)
}

func getTrash(db *sql.DB, id int) (models.Article, bool) {
	var a models.Article
	err := db.QueryRow(
		"SELECT "+articleCols+" FROM trash WHERE id = ?", id,
	).Scan(&a.ID, &a.Title, &a.Content, &a.Category, &a.Status)
	if err == sql.ErrNoRows {
		return a, false
	}
	return a, err == nil
}

// RestoreTrash moves a trashed article back into the articles table
// (status reset to 'draft') and removes it from trash, in one transaction.
func RestoreTrash(c *gin.Context, db *sql.DB) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "id must be an integer"})
		return
	}
	a, ok := getTrash(db, id)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"message": "trashed article not found"})
		return
	}
	tx, err := db.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}
	if _, err := tx.Exec(
		"INSERT INTO articles (title, content, category, status) VALUES (?, ?, ?, 'draft')",
		a.Title, a.Content, a.Category,
	); err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}
	if _, err := tx.Exec("DELETE FROM trash WHERE id = ?", id); err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}
	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "restored from trash"})
}

// DeleteTrash permanently removes a trashed article (hard delete from trash).
func DeleteTrash(c *gin.Context, db *sql.DB) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "id must be an integer"})
		return
	}
	res, err := db.Exec("DELETE FROM trash WHERE id = ?", id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		c.JSON(http.StatusNotFound, gin.H{"message": "trashed article not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "permanently deleted"})
}
