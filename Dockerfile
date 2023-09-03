FROM golang:1.20.6

# Set destination for COPY
WORKDIR /CloudMigrationApp

# Download Go modules
COPY go.mod go.sum ./
RUN go mod download

# Copy the source code. Note the slash at the end, as explained in
# https://docs.docker.com/engine/reference/builder/#copy
COPY *.go ./

# Build

RUN CGO_ENABLED=0 GOOS=linux go build -o CloudMigrationApp .
#RUN CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o CloudMigrationApp.exe .

# Run
CMD ./CloudMigrationApp
#FROM golang:1.20.6 AS build-stage

#WORKDIR /CloudMigrationApp

#COPY go.mod go.sum ./

#RUN go mod download

#COPY *.go .env ./

#RUN ls

#RUN CGO_ENABLED=0 GOOS=linux go build -o CloudMigrationApp

#FROM gcr.io/distroless/base-debian11 AS build-release-stage

#WORKDIR /CloudMigrationApp

#COPY --from=build-stage /CloudMigrationApp/CloudMigrationApp ./

#RUN chmod 0755 CloudMigrationApp

#CMD ["/bin/bash", "-c", "chmod", "/CloudMigrationApp"]