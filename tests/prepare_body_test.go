package tests

import (
	"gitlab.com/stihi/stihi-backend/app/filters"
	"strings"
	"testing"
)

func TestSetImgForURLInBody(t *testing.T) {
	text := "http://sample.com/ahhghg/ajh.jpg"
	res := filters.HTMLBodyFilter(text)
	if res != `<img src="https://sample.com/ahhghg/ajh.jpg" class="backend_processed_image"/>` {
		t.Error("Неверный текст при замене URL картинки на html тэг если ссылка это весь текст")
	}

	text = "http://imgp.golos.io/0x0/https://i.imgur.com/9ZebFvu.jpg"
	res = filters.HTMLBodyFilter(text)
	if res != `<img src="https://imgp.golos.io/0x0/https://i.imgur.com/9ZebFvu.jpg" class="backend_processed_image"/>` {
		t.Error("Неверный текст при замене URL реальной картинки на html тэг если ссылка это весь текст")
	}

	text = "Тестовый текст https://sample.com/ahhghg/ajh.jpg"
	res = filters.HTMLBodyFilter(text)
	if res != `Тестовый текст <img src="https://sample.com/ahhghg/ajh.jpg" class="backend_processed_image"/>` {
		t.Error("Неверный текст при замене URL картинки на html тэг в конце текста")
	}

	text = "Тестовый текст https://sample.com/ahhghg/ajh.jpg лоролролр"
	res = filters.HTMLBodyFilter(text)
	if res != `Тестовый текст <img src="https://sample.com/ahhghg/ajh.jpg" class="backend_processed_image"/> лоролролр` {
		t.Error("Неверный текст при замене URL картинки на html тэг в середине текста")
	}

	text = "https://sample.com/ahhghg/ajh.jpg лоролролр"
	res = filters.HTMLBodyFilter(text)
	if res != `<img src="https://sample.com/ahhghg/ajh.jpg" class="backend_processed_image"/> лоролролр` {
		t.Error("Неверный текст при замене URL картинки на html тэг в начале текста")
	}

	text = "https://sample.com/ahhghg/ajh1.jpg\nТестовый текст https://sample.com/ahhghg/ajh2.jpg лоролролр\nhttps://sample.com/ahhghg/ajh3.jpg"
	res = filters.HTMLBodyFilter(text)
	if res != `<img src="https://sample.com/ahhghg/ajh1.jpg" class="backend_processed_image"/>
Тестовый текст <img src="https://sample.com/ahhghg/ajh2.jpg" class="backend_processed_image"/> лоролролр
<img src="https://sample.com/ahhghg/ajh3.jpg" class="backend_processed_image"/>` {
		t.Error("Неверный текст при замене URL картинки на html тэг в начале, середине и конце текста")
	}

	text = "<p>https://sample.com/ahhghg/ajh1.jpg</p>"
	res = filters.HTMLBodyFilter(text)
	if res != `<p><img src="https://sample.com/ahhghg/ajh1.jpg" class="backend_processed_image"/></p>` {
		t.Error("Неверный текст при замене URL картинки на html тэг между тэгами")
	}
}

func TestSetYoutubeForURLInBody(t *testing.T) {
	// В конце текста
	text := "Тестовый текст https://www.youtube.com/watch?v=w9JrUVI_h4I"
	res := filters.HTMLBodyFilter(text)
	if res != `Тестовый текст <iframe title="YouTube video player" class="backend_processed_video" src="https://www.youtube.com/embed/w9JrUVI_h4I" frameborder="0" allowfullscreen="1"></iframe>` {
		t.Error("Неверный текст при замене обычного URL youtube видео на html тэг в конце текста")
	}

	text = "Тестовый текст https://youtu.be/w9JrUVI_h4I"
	res = filters.HTMLBodyFilter(text)
	if res != `Тестовый текст <iframe title="YouTube video player" class="backend_processed_video" src="https://www.youtube.com/embed/w9JrUVI_h4I" frameborder="0" allowfullscreen="1"></iframe>` {
		t.Error("Неверный текст при замене URL youtu.be видео на html тэг в конце текста")
	}

	// В середине текста
	text = "Тестовый текст https://www.youtube.com/watch?v=w9JrUVI_h4I - классное видео"
	res = filters.HTMLBodyFilter(text)
	if res != `Тестовый текст <iframe title="YouTube video player" class="backend_processed_video" src="https://www.youtube.com/embed/w9JrUVI_h4I" frameborder="0" allowfullscreen="1"></iframe> - классное видео` {
		t.Error("Неверный текст при замене обычного URL youtube видео на html тэг в середине текста")
	}

	text = "Тестовый текст https://youtu.be/w9JrUVI_h4I - классное видео"
	res = filters.HTMLBodyFilter(text)
	if res != `Тестовый текст <iframe title="YouTube video player" class="backend_processed_video" src="https://www.youtube.com/embed/w9JrUVI_h4I" frameborder="0" allowfullscreen="1"></iframe> - классное видео` {
		t.Error("Неверный текст при замене URL youtu.be видео на html тэг в середине текста")
	}

	// В начале текста
	text = "https://www.youtube.com/watch?v=w9JrUVI_h4I - классное видео"
	res = filters.HTMLBodyFilter(text)
	if res != `<iframe title="YouTube video player" class="backend_processed_video" src="https://www.youtube.com/embed/w9JrUVI_h4I" frameborder="0" allowfullscreen="1"></iframe> - классное видео` {
		t.Error("Неверный текст при замене обычного URL youtube видео на html тэг в начале текста")
	}

	text = "https://www.youtube.com/watch?v=w9JrUVI_h4I&t=10s - классное видео"
	res = filters.HTMLBodyFilter(text)
	if res != `<iframe title="YouTube video player" class="backend_processed_video" src="https://www.youtube.com/embed/w9JrUVI_h4I" frameborder="0" allowfullscreen="1"></iframe> - классное видео` {
		t.Error("Неверный текст при замене обычного URL youtube видео с временной меткой на html тэг в начале текста")
	}

	text = "https://youtu.be/w9JrUVI_h4I - классное видео"
	res = filters.HTMLBodyFilter(text)
	if res != `<iframe title="YouTube video player" class="backend_processed_video" src="https://www.youtube.com/embed/w9JrUVI_h4I" frameborder="0" allowfullscreen="1"></iframe> - классное видео` {
		t.Error("Неверный текст при замене URL youtu.be видео на html тэг в начале текста")
	}

	text = "https://youtu.be/w9JrUVI_h4I?t=10s - классное видео"
	res = filters.HTMLBodyFilter(text)
	if res != `<iframe title="YouTube video player" class="backend_processed_video" src="https://www.youtube.com/embed/w9JrUVI_h4I" frameborder="0" allowfullscreen="1"></iframe> - классное видео` {
		t.Error("Неверный текст при замене URL youtu.be видео с временной меткой на html тэг в начале текста")
	}

	text = "<p>https://youtu.be/w9JrUVI_h4I?t=10s</p>"
	res = filters.HTMLBodyFilter(text)
	if res != `<p><iframe title="YouTube video player" class="backend_processed_video" src="https://www.youtube.com/embed/w9JrUVI_h4I" frameborder="0" allowfullscreen="1"></iframe></p>` {
		t.Error("Неверный текст при замене URL youtu.be видео с временной меткой на html тэг между тэгами")
	}

	text = `https://www.youtube.com/watch?v=EaHLac_PHQY&feature=youtu.be&list=RDEaHLac_PHQY`
	res = filters.HTMLBodyFilter(text)
	if res != `<iframe title="YouTube video player" class="backend_processed_video" src="https://www.youtube.com/embed/EaHLac_PHQY" frameborder="0" allowfullscreen="1"></iframe>` {
		t.Error("Неверный текст при замене URL youtube.com видео если в URL есть лишние параметры")
	}
}

func TestUserLinkInBody(t *testing.T) {
	text := "@user1-a"
	res := filters.HTMLBodyFilter(text)
	if res != `<a href="/@user1-a" class="user-link">@user1-a</a>` {
		t.Error("Неверный текст при замене ссылки на пользователя на html тэг если ссылка это весь текст")
	}

	text = "Тестовый текст @user1-a"
	res = filters.HTMLBodyFilter(text)
	if res != `Тестовый текст <a href="/@user1-a" class="user-link">@user1-a</a>` {
		t.Error("Неверный текст при замене ссылки на пользователя на html тэг в конце текста")
	}

	text = "Тестовый текст @user1-a лоролролр"
	res = filters.HTMLBodyFilter(text)
	if res != `Тестовый текст <a href="/@user1-a" class="user-link">@user1-a</a> лоролролр` {
		t.Error("Неверный текст при замене ссылки на пользователя на html тэг в середине текста")
	}

	text = "@user1-a лоролролр"
	res = filters.HTMLBodyFilter(text)
	if res != `<a href="/@user1-a" class="user-link">@user1-a</a> лоролролр` {
		t.Error("Неверный текст при замене ссылки на пользователя на html тэг в начале текста")
	}

	text = "@user1-a Тестовый текст @user2-b лоролролр @user3-c"
	res = filters.HTMLBodyFilter(text)
	if res != `<a href="/@user1-a" class="user-link">@user1-a</a> Тестовый текст <a href="/@user2-b" class="user-link">@user2-b</a> лоролролр <a href="/@user3-c" class="user-link">@user3-c</a>` {
		t.Error("Неверный текст при замене ссылки на пользователя на html тэг в начале, середине и конце текста")
	}

	text = "<p>@user1-a</p>"
	res = filters.HTMLBodyFilter(text)
	if res != `<p><a href="/@user1-a" class="user-link">@user1-a</a></p>` {
		t.Error("Неверный текст при замене ссылки на пользователя на html тэг между тэгами")
	}

	text = "\n@user1-a\n"
	res = filters.HTMLBodyFilter(text)
	if res != `<a href="/@user1-a" class="user-link">@user1-a</a>`+"\n" {
		t.Error("Неверный текст при замене ссылки на пользователя на html тэг с переносом строк")
	}

	text = "\n@user1-a,"
	res = filters.HTMLBodyFilter(text)
	if res != `<a href="/@user1-a" class="user-link">@user1-a</a>`+"," {
		t.Error("Неверный текст при замене ссылки на пользователя на html тэг с переносом строки и запятой")
	}

	text = `<a href="/@user1-a" class="user-link">@user1-a</a>`
	res = filters.HTMLBodyFilter(text)
	if res != `<a href="/@user1-a" class="user-link">@user1-a</a>` {
		t.Error("Неверный текст при замене существующей html ссылки на пользователя на html тэг")
	}

	text = "Раз два три <a href=\"/@user1-a\" class=\"user-link\">@user1-a</a> четыре пять"
	res = filters.HTMLBodyFilter(text)
	if res != `Раз два три <a href="/@user1-a" class="user-link">@user1-a</a> четыре пять` {
		t.Error("Неверный текст при замене существующей html ссылки на пользователя на html тэг по середине текста")
	}

	text = "\n<a href=\"/@user1-a\" class=\"user-link\">@user1-a</a>\n"
	res = filters.HTMLBodyFilter(text)
	if res != `<a href="/@user1-a" class="user-link">@user1-a</a>`+"\n" {
		t.Error("Неверный текст при замене существующей html ссылки на пользователя на html тэг между переводами строк")
	}

	text = "<p><a href=\"/@user1-a\" class=\"user-link\">@user1-a</a></p>"
	res = filters.HTMLBodyFilter(text)
	if res != `<p><a href="/@user1-a" class="user-link">@user1-a</a></p>` {
		t.Error("Неверный текст при замене существующей html ссылки на пользователя на html тэг между тэгами")
	}

	text = "<a href=\"/@user1-a\" class=\"user-link\">oiuiou</a>"
	res = filters.HTMLBodyFilter(text)
	if res != `<a href="/@user1-a" class="user-link">oiuiou</a>` {
		t.Error("Неверный текст при отсутствии замены имени пользователя внутри тэга")
	}

	text = "http://site.ru/@user1-a/list"
	res = filters.HTMLBodyFilter(text)
	if res != `http://site.ru/@user1-a/list` {
		t.Error("Неверный текст при замене существующей html ссылки на пользователя в составе url")
	}

	text = "Тест @user1-a\nТекст\nТекст 2\n<center>центр</center>\nТекст 3"
	res = filters.HTMLBodyFilter(text)
	if res != "Тест <a href=\"/@user1-a\" class=\"user-link\">@user1-a</a>\nТекст\nТекст 2\n<center>центр</center>\nТекст 3" {
		t.Error("Неверный текст при замене линка пользователя и наличии в тексте html тэгов")
	}

	text = "*Дизайн @konti*\n\nТекст 1\n<br/>\nТекст 2\n\n<center>[![](https://s19.postimg.cc/7dubqikrn/stihi-io650.jpg)](https://golos.io/ru--delegaty/@stihi-io/delegat-stihi-io) </center>\n"
	res = filters.HTMLBodyFilter(text)
	if res != "*Дизайн <a href=\"/@konti\" class=\"user-link\">@konti</a>*\n\nТекст 1\n<br/>\nТекст 2\n\n<center>[![](https://s19.postimg.cc/7dubqikrn/stihi-io650.jpg)](https://golos.io/ru--delegaty/@stihi-io/delegat-stihi-io) </center>\n" {
		t.Error("Неверный текст при замене линка пользователя и наличии в тексте * после имени пользователя")
	}

	text = "@tusechka<br>@marfa"
	res = filters.HTMLBodyFilter(text)
	if res != `<a href="/@tusechka" class="user-link">@tusechka</a><br/><a href="/@marfa" class="user-link">@marfa</a>` {
		t.Error("Неверный текст при замене линка пользователя и наличии в тексте <br> между именами пользователей")
	}
}

func TestRemoveScriptsInBody(t *testing.T) {
	text := `Текст тела <script>
какой-то скрипт;
run_bad_function();
</script> орпропро`
	res := filters.HTMLBodyFilter(text)
	if strings.Contains(res, "<script") || strings.Contains(res, "</script") {
		t.Error("Неправильное обезвреживание скриптов в контенте")
	}
}

func TestRemoveMarkdownInPreview(t *testing.T) {
	text := `![](image.jpg) Текст превью.`
	res := filters.HTMLPreviewFilter(text, 10)
	if res != "Текст прев" {
		t.Error("Неверный текст при замене маркдауна в превью")
	}
}