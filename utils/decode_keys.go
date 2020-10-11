package main

import (
	"fmt"
	"gitlab.com/stihi/stihi-backend/blockchain"
	"os"


)

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Command:\n\ndecode_keys <golos login> <password|private key>\n")
		os.Exit(1)
	}

	login := os.Args[1]
	pass := os.Args[2]

	if pass[0] == 'P' {
		postingP	:= blockchain.GetPrivateKey(login, "posting", pass)
		activeP		:= blockchain.GetPrivateKey(login, "active", pass)
		ownerP 		:= blockchain.GetPrivateKey(login, "owner", pass)
		memoP		:= blockchain.GetPrivateKey(login, "memo_key", pass)

		fmt.Printf("PVT Posting: %s\nPVT Active: %s\nPVT Owner: %s\nPVT Memo: %s\n\n",
			postingP, activeP, ownerP, memoP)

		posting 	:= blockchain.GetPublicKey("GLS", postingP)
		active 		:= blockchain.GetPublicKey("GLS", activeP)
		owner 		:= blockchain.GetPublicKey("GLS", ownerP)
		memo 		:= blockchain.GetPublicKey("GLS", memoP)

		fmt.Printf("PUB Posting: %s\nPUB Active: %s\nPUB Owner: %s\nPUB Memo: %s\n\n",
			posting, active, owner, memo)
	} else {
		pub 		:= blockchain.GetPublicKey("GLS", pass)
		fmt.Printf("PUB key: %s\n\n", pub)
	}
}