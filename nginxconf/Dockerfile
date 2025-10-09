FROM tiangolo/nginx-rtmp:latest

# Copy custom nginx config
# COPY ./nginx.conf /etc/nginx/nginx.conf
# COPY ./stat.xsl /etc/nginx/stat.xsl

# Ensure /tmp/hls and /tmp/dash exist and are writable
RUN groupadd -r nginx && useradd -r -g nginx nginx
RUN mkdir -p /tmp/hls /tmp/dash && chown -R nginx:nginx /tmp/hls /tmp/dash

EXPOSE 1935 8080

CMD ["nginx", "-g", "daemon off;"]