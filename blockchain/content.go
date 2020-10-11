package blockchain

import (
	"gitlab.com/stihi/stihi-backend/app/random"
	"gitlab.com/stihi/stihi-backend/blockchain/translit"
	"strings"
)

func GenPermlink(prefix, suffix, author, title string) string {
	encTitle, _ := translit.EncodeTitle(title)
	permlink := prefix + author + encTitle + suffix
	permlink = strings.Replace(permlink, ".", "-", -1)

	if len(permlink) > 255 {
		randStr := random.StringWithCharset(32, random.CharsetAlD, random.CharsetAlDLen)
		permlink = permlink[:210] + randStr
	}

	return permlink
}
