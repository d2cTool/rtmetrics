KEYS_DIR := keys

.PHONY: gen-keys gen-certs
gen-keys:
	mkdir -p $(KEYS_DIR)
	openssl genrsa -out $(KEYS_DIR)/private.pem 4096
	openssl rsa -in $(KEYS_DIR)/private.pem -pubout -out $(KEYS_DIR)/public.pem

gen-certs:
	mkdir -p $(KEYS_DIR)
	openssl req -x509 -newkey rsa:4096 -sha256 -days 365 -nodes \
		-keyout $(KEYS_DIR)/grpc.key -out $(KEYS_DIR)/grpc.crt \
		-subj "/CN=localhost" \
		-addext "subjectAltName=DNS:localhost,IP:127.0.0.1,IP:::1"
