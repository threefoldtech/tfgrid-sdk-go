DIRS := "activation-service" "farmerbot" "grid-cli" "grid-client" "grid-proxy" "gridify" "monitoring-bot" "rmb-sdk-go" "user-contracts-mon" 

release-rmb:
	@echo "Release RMB..." 
	git tag -a "rmb-sdk-go/${VERSION}" -m "release rmb-sdk-go/${VERSION}" && \
  git push origin rmb-sdk-go/${VERSION}

release:
	@echo "Running release script..." 
	chmod +x release.sh 
	./release.sh

lint:
	for DIR in ${DIRS} ; do \
		cd $$DIR && golangci-lint run -c ../.golangci.yml --timeout 10m && cd ../ ; \
	done

tidy:
	go work sync
	for DIR in ${DIRS} ; do \
		cd $$DIR && go mod tidy && cd ../ ; \
	done
