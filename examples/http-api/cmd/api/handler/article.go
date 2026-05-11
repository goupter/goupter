package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/goupter/goupter/examples/http-api/model"
	"github.com/goupter/goupter/examples/http-api/pkg/token"
	"github.com/goupter/goupter/pkg/cache"
	"github.com/goupter/goupter/pkg/errors"
	"github.com/goupter/goupter/pkg/response"
)

type ArticleHandler struct {
	articleModel *model.ArticlesModel
	cache        cache.Cache
}

func NewArticleHandler(articleModel *model.ArticlesModel, cache cache.Cache) *ArticleHandler {
	return &ArticleHandler{
		articleModel: articleModel,
		cache:        cache,
	}
}

type CreateArticleRequest struct {
	Title   string `json:"title" binding:"required"`
	Content string `json:"content"`
}

type UpdateArticleRequest struct {
	Title   *string `json:"title,omitempty"`
	Content *string `json:"content,omitempty"`
	Status  *int8   `json:"status,omitempty"`
}

type ArticleResponse struct {
	ID        int64  `json:"id"`
	Title     string `json:"title"`
	Content   string `json:"content"`
	AuthorID  int64  `json:"author_id"`
	Status    int8   `json:"status"`
	ViewCount int32  `json:"view_count"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

func toArticleResponse(a *model.Articles) *ArticleResponse {
	return &ArticleResponse{
		ID:        a.ID,
		Title:     a.Title,
		Content:   a.Content,
		AuthorID:  a.AuthorID,
		Status:    a.Status,
		ViewCount: a.ViewCount,
		CreatedAt: a.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt: a.UpdatedAt.Format("2006-01-02 15:04:05"),
	}
}

// Create creates a new article (using WrapJSONUser)
func (h *ArticleHandler) Create(c *gin.Context, user *token.UserInfo, req *CreateArticleRequest) {
	article := &model.Articles{
		Title:    req.Title,
		Content:  req.Content,
		AuthorID: user.UserId,
		Status:   1,
	}

	if err := h.articleModel.Insert(c.Request.Context(), nil, article); err != nil {
		response.Error(c, errors.Wrap(err, errors.CodeInternalError, "failed to create article"))
		return
	}

	response.Success(c, toArticleResponse(article))
}

// Get retrieves an article by ID
func (h *ArticleHandler) Get(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, errors.New(errors.CodeBadRequest, "invalid id"))
		return
	}

	article, err := h.articleModel.FindByID(c.Request.Context(), id)
	if err != nil {
		response.Error(c, errors.New(errors.CodeNotFound, "article not found"))
		return
	}

	// Increment view count asynchronously
	go func() { _ = h.articleModel.IncrViewCount(c.Request.Context(), id) }()

	response.Success(c, toArticleResponse(article))
}

// List returns paginated articles (using WrapUserPaging)
func (h *ArticleHandler) List(c *gin.Context, user *token.UserInfo, page, pageSize int) {
	articles, total, err := h.articleModel.FindPage(c.Request.Context(), page, pageSize, "id DESC", "")
	if err != nil {
		response.Error(c, errors.Wrap(err, errors.CodeInternalError, "failed to list articles"))
		return
	}

	list := make([]*ArticleResponse, len(articles))
	for i := range articles {
		list[i] = toArticleResponse(articles[i])
	}

	response.Success(c, gin.H{
		"list":     list,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
}

// Update updates an article (using WrapJSONUser)
func (h *ArticleHandler) Update(c *gin.Context, user *token.UserInfo, req *UpdateArticleRequest) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, errors.New(errors.CodeBadRequest, "invalid id"))
		return
	}

	article, err := h.articleModel.FindByID(c.Request.Context(), id)
	if err != nil {
		response.Error(c, errors.New(errors.CodeNotFound, "article not found"))
		return
	}

	if req.Title != nil {
		article.Title = *req.Title
	}
	if req.Content != nil {
		article.Content = *req.Content
	}
	if req.Status != nil {
		article.Status = *req.Status
	}

	if err := h.articleModel.Update(c.Request.Context(), nil, article); err != nil {
		response.Error(c, errors.Wrap(err, errors.CodeInternalError, "failed to update article"))
		return
	}

	response.Success(c, toArticleResponse(article))
}

// Delete deletes an article (using WrapUser)
func (h *ArticleHandler) Delete(c *gin.Context, user *token.UserInfo) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, errors.New(errors.CodeBadRequest, "invalid id"))
		return
	}

	if err := h.articleModel.Delete(c.Request.Context(), nil, map[string]any{"id": id}); err != nil {
		response.Error(c, errors.Wrap(err, errors.CodeInternalError, "failed to delete article"))
		return
	}

	response.Success(c, gin.H{"deleted": true})
}
