FROM docker.friedmicro-lab.org/base AS build

WORKDIR /app
COPY go.mod main.go ./
ENV CGO_ENABLED=0 GOTOOLCHAIN=local
RUN go build -trimpath -ldflags="-s -w" -o /opt/cellnotify .

FROM docker.friedmicro-lab.org/base-slim
COPY --from=build /opt/cellnotify /opt/cellnotify

CMD ["/opt/cellnotify"]
