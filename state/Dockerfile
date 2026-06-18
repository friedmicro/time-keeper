FROM docker.friedmicro-lab.org/base

WORKDIR /app
COPY go.mod main.go ./
ENV CGO_ENABLED=0 GOTOOLCHAIN=local
RUN go build -trimpath -ldflags="-s -w" -o timestate .
EXPOSE 5000
CMD ["./timestate"]
