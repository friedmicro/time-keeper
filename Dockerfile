FROM docker.friedmicro-lab.org/base AS build

COPY frontend frontend
RUN cd frontend; npm ci; npm run build; mv dist /opt/
COPY api api
RUN cd api; go mod download; CGO_ENABLED=0 GOOS=linux go build -o /opt/timekeeper-api .

FROM docker.friedmicro-lab.org/base-nginx
COPY frontend/nginx.conf /etc/nginx/
COPY --from=build /opt/timekeeper-api /opt/timekeeper-api
COPY --from=build /opt/dist/ /htmlcontent/


EXPOSE 5005
CMD ["/bin/sh", "-c", "/opt/timekeeper-api & exec nginx -g \"daemon off;\""]
