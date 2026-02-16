

.PHONY: frontend backend electron-install electron-dev electron-build electron-install-mac electron-build-win

frontend:
	cd web && npm install && npm run dev

backend:
	go build -o robodiff && ./robodiff --dir /tmp/robot_results --addr :8080

electron-install:
	npm install
	npm --prefix web install

electron-dev: electron-install
	npm run dev

electron-build: electron-install
	npm run build

electron-install-mac: electron-build
	@echo "Installing app to /Applications"
	@APP=$$(find dist -maxdepth 3 -name "*.app" -print -quit); \
	if [ -z "$$APP" ]; then \
		echo "No .app found under dist/. Build may have failed."; \
		exit 1; \
	fi; \
	cp -R "$$APP" /Applications/

electron-build-win: electron-install
	@echo "Building Windows NSIS installer"
	npm run build -- --win nsis