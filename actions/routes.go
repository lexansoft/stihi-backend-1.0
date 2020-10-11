package actions

import (
	"fmt"
	"github.com/dchest/captcha"
	"gitlab.com/stihi/stihi-backend/app"
	"gitlab.com/stihi/stihi-backend/app/config"
	"net/http"
)

var Config *config.BackendConfig

func InitRoutes(config *config.BackendConfig) {
	Config = config

	ver := "v2"

	// Admin API
	http.HandleFunc("/api/"+ver+"/get_info", GetContentInfo)

	// Fix pages
	http.HandleFunc("/api/"+ver+"/update_fix_page", UpdateFixPage)
	http.HandleFunc("/api/"+ver+"/get_fix_page", GetFixPage)
	http.HandleFunc("/api/"+ver+"/get_fix_pages_list", GetFixPagesList)

	// Content API
	http.HandleFunc("/api/"+ver+"/get_articles_list", GetArticlesList)
	http.HandleFunc("/api/"+ver+"/get_article", GetArticle)
	http.HandleFunc("/api/"+ver+"/get_comments_list", GetCommentsList)
	http.HandleFunc("/api/"+ver+"/get_comment", GetComment)
	http.HandleFunc("/api/"+ver+"/get_user_comments_list", GetUserCommentsList)
	http.HandleFunc("/api/"+ver+"/get_all_comments_list", GetAllCommentsList)
	http.HandleFunc("/api/"+ver+"/get_votes_list", GetVotesList)
	http.HandleFunc("/api/"+ver+"/get_rubrics_list", GetRubricsList)
	http.HandleFunc("/api/"+ver+"/get_exchange_rates", GetExchangeRates)

	// Posting API
	http.HandleFunc("/api/"+ver+"/create_article", CreateArticle)
	http.HandleFunc("/api/"+ver+"/update_article", UpdateArticle)
	http.HandleFunc("/api/"+ver+"/create_comment", CreateComment)
	http.HandleFunc("/api/"+ver+"/update_comment", UpdateComment)
	http.HandleFunc("/api/"+ver+"/create_vote", CreateVote)
	http.HandleFunc("/api/"+ver+"/delete_content", DeleteContent)

	// Login API
	http.HandleFunc("/api/"+ver+"/login", Login)
	http.HandleFunc("/api/"+ver+"/signup", Signup)
	http.HandleFunc("/api/"+ver+"/new_password", GeneratePassword)

	// Users Api
	http.HandleFunc("/api/"+ver+"/get_user_info", GetUserInfo)
	http.HandleFunc("/api/"+ver+"/update_user_info", UpdateUserInfo)
	http.HandleFunc("/api/"+ver+"/get_users_list", GetUsersList)
	http.HandleFunc("/api/"+ver+"/get_user_tags_list", GetUserTagsList)
	http.HandleFunc("/api/"+ver+"/get_user_battery", GetUserBattery)
	http.HandleFunc("/api/"+ver+"/get_user_period_leader", GetUsersPeriodLeader)

	// Follow API
	http.HandleFunc("/api/"+ver+"/get_user_subscriptions_list", GetUserSubscriptionsList)
	http.HandleFunc("/api/"+ver+"/get_user_subscribers_list", GetUserSubscribersList)
	http.HandleFunc("/api/"+ver+"/user_subscribe", UserSubscribe)
	http.HandleFunc("/api/"+ver+"/user_unsubscribe", UserUnsubscribe)
	http.HandleFunc("/api/"+ver+"/user_ignore", UserIgnore)
	http.HandleFunc("/api/"+ver+"/user_unignore", UserUnignore)

	// Announces API
	http.HandleFunc("/api/"+ver+"/get_announce_pages_list", GetAnnouncePagesList)
	http.HandleFunc("/api/"+ver+"/create_announce", CreateAnnounce)
	http.HandleFunc("/api/"+ver+"/get_announces_list", GetAnnouncesList)

	// Invite API
	http.HandleFunc("/api/"+ver+"/create_invite", CreateInvite)
	http.HandleFunc("/api/"+ver+"/get_invites_list", GetInvitesList)

	// Wallet API
	http.HandleFunc("/api/"+ver+"/send_tokens", WalletSendTokens)
	http.HandleFunc("/api/"+ver+"/send_golos_to_power", WalletConvertGolosToPower)
	http.HandleFunc("/api/"+ver+"/send_power_to_golos", WalletConvertPowerToGolos)
	http.HandleFunc("/api/"+ver+"/show_private_keys", WalletShowPrivateKeys)
	http.HandleFunc("/api/"+ver+"/get_withdraw_info", WalletGetWithdrawInfo)
	http.HandleFunc("/api/"+ver+"/get_history", WalletGetHistory)
	http.HandleFunc("/api/"+ver+"/refresh_balance", WalletRefreshBalance)

	// Ban/Unban
	http.HandleFunc("/api/"+ver+"/ban_user", BanUser)
	http.HandleFunc("/api/"+ver+"/unban_user", UnbanUser)
	http.HandleFunc("/api/"+ver+"/ban_content", BanContent)
	http.HandleFunc("/api/"+ver+"/unban_content", UnbanContent)

	// Captcha
	http.HandleFunc("/api/"+ver+"/get_captcha", NewCaptcha)
	http.Handle("/api/"+ver+"/captcha/", captcha.Server(captcha.StdWidth, captcha.StdHeight))

	// Sharpay
	http.HandleFunc("/api/"+ver+"/sharpay/get_share_count", SharepayGetShareCount)

	// Static pages
	http.HandleFunc("/static/article", StaticArticle)

	// Error handler
	http.HandleFunc("/", errorHandler)
}

func errorHandler(w http.ResponseWriter, r *http.Request) {
	clientIp := GetRealClientIP(r)
	if clientIp == "" {
		clientIp = r.RemoteAddr
	}
	app.Info.Printf("Wrong request to '%s' from %s - 404", r.URL.String(), clientIp)
	w.WriteHeader(http.StatusNotFound)
	_, _ = fmt.Fprint(w, "404 - not found")
}
