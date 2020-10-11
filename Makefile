.PHONY: vendor vendor_add migrations templates

export CGO_ENABLED=0

BUILD := $(shell cat /tmp/build)

all: sb bk

migrations:
	resources -declare -var=List -package=migrations -output=migrations/migrations.go migrations/*.sql
templates:
	resources -declare -var=List -package=templates -output=templates/templates.go templates/*.tmpl
sb: utils/scan_blockchain.go app/*/*
	CGO_ENABLED=1 go build -race utils/scan_blockchain.go
sb_run: sb scan_blockchain
	./scan_blockchain -db_config=/home/andy/projects/STIHI.IO/stihi-backend/configs/db_sample.yaml -redis_config=/home/andy/projects/STIHI.IO/stihi-backend/configs/redis_sample.yaml -config=/home/andy/projects/STIHI.IO/stihi-backend/configs/scan_blockchain_config.yaml
sb_restart: sb scan_blockchain
	./scan_blockchain -db_config=/home/andy/projects/STIHI.IO/stihi-backend/configs/db_sample.yaml -redis_config=/home/andy/projects/STIHI.IO/stihi-backend/configs/redis_sample.yaml -config=/home/andy/projects/STIHI.IO/stihi-backend/configs/scan_blockchain_config.yaml restart
sb_start: sb_run
sb_status: sb scan_blockchain
	./scan_blockchain status
sb_stop: sb scan_blockchain
	./scan_blockchain stop
sb_production: utils/scan_blockchain.go app/*/*
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-X main.build=$(BUILD)" utils/scan_blockchain.go

bk: stihi_backend.go app/*/*
	CGO_ENABLED=1 go build -race stihi_backend.go
bk_run: bk stihi_backend
	./stihi_backend -db_config=/home/andy/projects/STIHI.IO/stihi-backend/configs/db_sample.yaml -redis_config=/home/andy/projects/STIHI.IO/stihi-backend/configs/redis_sample.yaml -config=/home/andy/projects/STIHI.IO/stihi-backend/configs/stihi_backend_config.yaml
bk_restart: bk stihi_backend
	./stihi_backend -db_config=/home/andy/projects/STIHI.IO/stihi-backend/configs/db_sample.yaml -redis_config=/home/andy/projects/STIHI.IO/stihi-backend/configs/redis_sample.yaml -config=/home/andy/projects/STIHI.IO/stihi-backend/configs/stihi_backend_config.yaml restart
bk_start: bk_run
bk_status: bk stihi_backend
	./stihi_backend status
bk_stop: bk stihi_backend
	./stihi_backend stop
bk_production: stihi_backend.go app/*/*
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-X main.build=$(BUILD)" stihi_backend.go

dk: utils/scan_blockchain.go app/*/*
	CGO_ENABLED=1 go build -race utils/decode_keys.go

bl: utils/blockchain_loader/* app/*/*
	CGO_ENABLED=1 go build -race utils/blockchain_loader/blockchain_loader.go
bl_production: utils/blockchain_loader/* app/*/*
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-X main.build=$(BUILD)" utils/blockchain_loader/blockchain_loader.go

blc: utils/blockchain_loader_cyberway/* app/*/*
	CGO_ENABLED=1 go build -race utils/blockchain_loader_cyberway/blockchain_loader_cyberway.go
blc_production: utils/blockchain_loader_cyberway/* app/*/*
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-X main.build=$(BUILD)" utils/blockchain_loader_cyberway/blockchain_loader_cyberway.go

sbc: utils/scan_blockchain_cyberway/scan_blockchain_cyberway.go app/*/*
	CGO_ENABLED=1 go build -race utils/scan_blockchain_cyberway/scan_blockchain_cyberway.go
sbc_production: utils/scan_blockchain_cyberway/scan_blockchain_cyberway.go app/*/*
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-X main.build=$(BUILD)" utils/scan_blockchain_cyberway/scan_blockchain_cyberway.go

bkc: stihi_backend_cyberway.go app/*/*
	CGO_ENABLED=1 go build -race stihi_backend_cyberway.go
bkc_production: stihi_backend_cyberway.go app/*/*
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-X main.build=$(BUILD)" stihi_backend_cyberway.go

ep: utils/events_processor/* app/*/*
	CGO_ENABLED=1 go build -o stihi_events_processor -race utils/events_processor/events_processor.go
ep_production: utils/events_processor/* app/*/*
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-X main.build=$(BUILD)" -o stihi_events_processor utils/events_processor/events_processor.go
