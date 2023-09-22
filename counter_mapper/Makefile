.PHONY: all docker clean

all:
	go build .

docker:
	docker build . -t "your-repo-name"/kubeedge-pi-counter-:v1.0.0

clean:
	rm -rf ./counter_mapper
