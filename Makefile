release-rmb:
	@echo "Release RMB..." 
	git tag -a "rmb-sdk-go/${VERSION}" -m "release rmb-sdk-go/${VERSION}" && \
  git push origin rmb-sdk-go/${VERSION}

release:
	@echo "Running release script..." 
	chmod +x release.sh 
	./release.sh

lint:
	for DIR in "activation-service" "grid-cli" "grid-client" "grid-proxy" "gridify" "monitoring-bot" "rmb-sdk-go" "user-contracts-mon" ; do \
		cd $$DIR && golangci-lint run -c ../.golangci.yml --timeout 10m && cd ../ ; \
	done
