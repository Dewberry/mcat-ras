FROM golang:1.20-alpine as build

RUN apk add --no-cache \
    pkgconfig \
    gcc \
    libc-dev \
    git \ 
    gdal-dev \ 
    geos-dev

RUN go install github.com/githubnemo/CompileDaemon@v1.4.0

COPY ./ /app
WORKDIR /app

RUN go build main.go

ENTRYPOINT CompileDaemon --build="go build main.go" --command=./main


# FROM osgeo/gdal:alpine-normal-3.6.3 as local
# COPY --from=build /app/main /app/main
# RUN wget https://github.com/HydrologicEngineeringCenter/hec-downloads/releases/download/1.0.23/HEC-RAS_62_Example_Projects.zip

# RUN unzip HEC-RAS_62_Example_Projects.zip
# RUN mkdir mcat-ras-testing
# RUN mv /Example_Projects/ /mcat-ras-testing
# RUN rm HEC-RAS_62_Example_Projects.zip

# ENTRYPOINT /app/main

FROM golang:1.20-alpine  as prod

RUN apk add --no-cache \
    pkgconfig \
    gdal-dev \ 
    geos-dev

COPY --from=build /app/main /app/main
ENTRYPOINT /app/main

