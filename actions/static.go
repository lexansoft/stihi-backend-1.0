package actions

import (
	"errors"
	"fmt"
	"gitlab.com/stihi/stihi-backend/app"
	"gitlab.com/stihi/stihi-backend/cache_level2"
	"gitlab.com/stihi/stihi-backend/templates"
	"html/template"
	"net/http"
	"strconv"
)

type ArticleContext struct {
	Article *cache_level2.Article
	URL     string
	Body	template.HTML
	Site	string
}

func StaticArticle(w http.ResponseWriter, r *http.Request) {
	ids, ok := r.URL.Query()["id"]
	if !ok || len(ids) < 1 {
		app.Error.Println(errors.New("No ID in request"))

		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, "404 - not found")
		return
	}

	id, err := strconv.ParseInt(ids[0], 10, 64)
	if err != nil {
		app.Error.Println(err)

		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, "Bad request")
		return
	}

	var context ArticleContext

	context.Article, err = DB.GetArticlePreview(id, false)
	if err != nil {
		app.Error.Println(err)

		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, "Bad request")
		return
	}

	// Определяем URL

	testPrefix := ""
	if Config.RPC.BlockchanName == "test" {
		testPrefix = "test."
	}
	context.URL = "https://"+testPrefix+"stihi.io/posts/"+ids[0]
	context.Site = testPrefix+"Stihi.io"

	if context.Article.User.NickName == "" {
		context.Article.User.NickName = context.Article.Author
	}

	/*
	if context.Article.Image == "" {
		context.Article.Image = "https://"+testPrefix+"stihi.io/frontend_assets_stihi/svg/logo.svg"
	}
	*/

	context.Body = template.HTML(context.Article.Body)

	tmplContent, ok := templates.List.String("/templates/article.tmpl")
	if !ok {
		app.Error.Println(err)

		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, "Bad template")
		return
	}

	t, err := template.New("article").Parse(tmplContent)
	if err != nil {
		app.Error.Println(err)

		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, "Bad template")
		return
	}

	err = t.Execute(w, context)
	if err != nil {
		app.Error.Println(err)

		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, "Bad template")
		return
	}
}
