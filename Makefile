release:
	@echo "Running release script..." 
	chmod +x release.sh 
	./release.sh

lint:
	for DIR in "activation-service" "grid-cli" "grid-client" "grid-proxy" "gridify" "monitoring-bot" "rmb-sdk-go" ; do \
		cd $$DIR && golangci-lint run -c ../.golangci.yml --timeout 10m && cd ../ ; \
	done
	