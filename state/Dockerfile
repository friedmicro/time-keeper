FROM docker.friedmicro-lab.org/base AS build

WORKDIR /app
COPY go.mod main.go ./
ENV CGO_ENABLED=0 GOTOOLCHAIN=local
RUN go build -trimpath -ldflags="-s -w" -o /opt/timestate .

FROM docker.friedmicro-lab.org/base-slim
COPY --from=build /opt/timestate /opt/timestate

EXPOSE 5000
CMD ["/opt/timestate"]
