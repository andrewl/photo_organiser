all: bin/photo_organiser bin-arm/photo_organiser

bin/photo_organiser : main.go
	go build -o bin/photo_organiser main.go

bin-arm/photo_organiser : main.go
	env GOOS=linux GOARCH=arm GOARM=5 go build -o bin-arm/photo_organiser
