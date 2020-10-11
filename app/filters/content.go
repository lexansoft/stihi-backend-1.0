package filters

import (
	"bytes"
	"encoding/xml"
	"gitlab.com/stihi/stihi-backend/app"
	htmlParser "golang.org/x/net/html"
	"golang.org/x/net/html/atom"
	"html"
	"io"
	"net/url"
	"regexp"
	"strings"
)

var (
	imgHtmlTagRegexp 		*regexp.Regexp
	youtubeHtmlTagRegexp 	*regexp.Regexp
	scriptHtmlTagRegexp 	*regexp.Regexp
	markdownImgLink			*regexp.Regexp
	replaceUserName			*regexp.Regexp
)

func init() {
	imgHtmlTagRegexp = regexp.MustCompile(`(?i)(\s|^|\<|\>|\n)(http|https):\/\/(.+)\.(jpg|jpeg|png|gif)(\s|$|\<|\>|\n)`)
	youtubeHtmlTagRegexp = regexp.MustCompile(
		`(\s|^|\<|\>|\n)http(|s):\/\/(www\.|)youtu(be\.com|\.be)\/(v\/|embed\/|watch\?|)(v=|)([^\&\?\s]+)([&|\?]{1}|&amp;|)[^\<\>\s]*(t=[^\<\>&\s]+|)(\s|\<|\>|$|\n|\r\n)`,
	)
	markdownImgLink = regexp.MustCompile(`\!\[[^\]]*\]\([^\)]*\)\s+`)
	scriptHtmlTagRegexp = regexp.MustCompile(
		`\<script(.*)\>|\<\/script(.*)\>`,
	)
	replaceUserName = regexp.MustCompile(`([^\/]|^)(\@[a-zA-Z0-9\-]+)`)
}

func IsImageURL(text string) bool {
	return imgHtmlTagRegexp.Match([]byte(text))
}

func HTMLBodyFilter(text string) string {
	str := text

	// Заменяем URL картинку на картинку в html тэге
	str = string(imgHtmlTagRegexp.ReplaceAll(
		[]byte(str),
		[]byte(`$1<img src="https://$3.$4" class="backend_processed_image"/>$5`)))

	// Заменяем ссылку на youtube на встроенное видео
	//str = string(youtubeHtmlTagRegexp.ReplaceAll(
	//	[]byte(str),
	//	[]byte(`$1<iframe title="YouTube video player" class="backend_processed_video" src="https://www.youtube.com/embed/$7" frameborder="0" allowfullscreen="1"></iframe>$10`)))
	str = ReplaceYoutubeURLToPlayer(str)

	str = ReplaceUserNameToLink(str)

	// Защита от вставки скриптов в тело поста или комментария
	str = string(scriptHtmlTagRegexp.ReplaceAll(
		[]byte(str),
		[]byte(``)))

	return str
	// parsed, _ := qvxBody.Parse(text)
	// return parsed
}

func HTMLPreviewFilter(text string, limit int) string {
	str := html.UnescapeString(text)
	str = RemoveHTMLTags(str)
	str = RemoveDupSpaces(str)

	// Удаляем URL картинки
	str = string(imgHtmlTagRegexp.ReplaceAll(
		[]byte(str),
		[]byte(``)))

	// Удаляем ссылку на youtube
	str = string(youtubeHtmlTagRegexp.ReplaceAll(
		[]byte(str),
		[]byte(``)))

	// Удаляем markdown
	str = string(markdownImgLink.ReplaceAll(
		[]byte(str),
		[]byte(``)))

	runes := []rune(str)
	if len(runes) >= limit {
		return string(runes[:limit])
	}
	return str
}

func TagCodeBuild(tag string, params map[string]string, content string) string {
	return "<pre><code>" + content + "<code><pre>\n"
}

func TagSharpBuild(str string) string {
	if matched, _ := regexp.MatchString(`^(?i)[\d\p{L}\_\-]{1,32}$`, str); !matched {
		return ""
	}
	return "<a href=\"/tags/" + url.QueryEscape(str) + "/\">#" + str + "</a>"
}

func TagAtBuild(str string) string {
	if matched, _ := regexp.MatchString(`^(?i)[\d\p{L}\_\-]{1,32}$`, str); !matched {
		return ""
	}
	return "<a href=\"/user/" + url.QueryEscape(str) + "/\">@" + str + "</a>"
}

func RemoveHTMLTags(str string) (string) {
	reg := regexp.MustCompile(`\<[^\<\>]+\>`)
	str = reg.ReplaceAllString(str, " ")

	return str
}

func RemoveDupSpaces(str string) (string) {
	// Удаление unbreakable spaces
	str = strings.Replace(str, "\u00A0", " ", -1)

	// Замена повторяющихся обычных пробелов на 1 пробел
	reg := regexp.MustCompile(`\s+`)
	str = reg.ReplaceAllString(str, " ")

	return str
}

func parseNode(node *htmlParser.Node, rx *regexp.Regexp, replace string) error {
	forRemove := make([]*htmlParser.Node, 0)
	for c := node.FirstChild; c != nil; c = c.NextSibling {
		switch c.Type {
		case htmlParser.TextNode:
			if node.Data == "a" {
				continue
			}

			// newStr := string(replaceUserName.ReplaceAll(
			//	[]byte(c.Data),
			//	[]byte(`$1<a href="/$2" class="user-link">$2</a>`)))
			newStr := string(rx.ReplaceAll(
				[]byte(c.Data),
				[]byte(replace)))

			if newStr != c.Data {
				reader := strings.NewReader(newStr)
				nodes, err := htmlParser.ParseFragment(reader, &htmlParser.Node{
					Type: htmlParser.ElementNode,
					Data: "body",
					DataAtom: atom.Body,
				})
				if err != nil {
					app.Error.Println(err)
					return err
				}

				parent := c.Parent
				for _, e := range nodes {
					// parent.AppendChild(e)
					parent.InsertBefore(e, c)
				}
				// parent.RemoveChild(c)
				forRemove = append(forRemove, c)
			}
		}
		err := parseNode(c, rx, replace)
		if err != nil {
			app.Error.Println(err)
			return err
		}
	}
	for _, c := range forRemove {
		c.Parent.RemoveChild(c)
	}

	return nil
}

type htmlXml struct {
	Body body `xml:"body"`
}
type body struct {
	Content string `xml:",innerxml"`
}

func parseHtml(htmlContent string, rx *regexp.Regexp, replace string) (string, error) {
	r := strings.NewReader(htmlContent)
	p, _ := htmlParser.Parse(r)
	err := parseNode(p, rx, replace)
	if err != nil {
		app.Error.Println(err)
		return "", err
	}
	var buf bytes.Buffer
	writer := io.Writer(&buf)
	htmlParser.Render(writer, p)

	// Извлекаем только body
	content := buf.String()
	h := htmlXml{}
	err = xml.NewDecoder(bytes.NewBufferString(content)).Decode(&h)
	if err != nil {
		app.Error.Println(err)
		return "", err
	}

	return h.Body.Content, nil
}

func ReplaceUserNameToLink(content string) string {
	res, err := parseHtml(content, replaceUserName, `$1<a href="/$2" class="user-link">$2</a>`)
	if err != nil {
		app.Error.Println(err)
		return content
	}
	return res
}

func ReplaceYoutubeURLToPlayer(content string) string {
	res, err := parseHtml(content, youtubeHtmlTagRegexp, `$1<iframe title="YouTube video player" class="backend_processed_video" src="https://www.youtube.com/embed/$7" frameborder="0" allowfullscreen="1"></iframe>$10`)
	if err != nil {
		app.Error.Println(err)
		return content
	}
	return res
}