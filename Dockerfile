FROM docker.friedmicro-lab.org/base

RUN apt install nginx -y
COPY frontend/nginx.conf /etc/nginx/conf.d/default.conf

COPY frontend frontend
RUN cd frontend; npm ci; npm run build; mv dist/* /usr/share/nginx/html/
COPY api api
RUN cd api; go mod download; CGO_ENABLED=0 GOOS=linux go build -o ./timekeeper-api .; mv ./timekeeper-api /usr/local/bin/timekeeper-api

CMD ["/bin/sh", "-c", "/usr/local/bin/timekeeper-api & exec nginx -g \"daemon off;\""]

EXPOSE 5005
