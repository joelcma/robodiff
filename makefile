

frontend:
	cd frontend && npm install && npm run dev

backend:
	go build -o robotdiff && ./robotdiff --dir /tmp/robot_results --addr :8080