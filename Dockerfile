FROM alpine:latest

COPY ./chat_server /chat_server
RUN chmod +x /chat_server

ENTRYPOINT ["/chat_server"]

EXPOSE 80 443 9000
CMD ["/chat_server"]