docker:
	docker build . -t catnipbrewer/teapot-server:latest
	docker push catnipbrewer/teapot-server:latest