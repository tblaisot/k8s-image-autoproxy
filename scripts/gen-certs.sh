#! /bin/bash

WEBHOOK_NS=k8s-image-autoproxy
WEBHOOK_NAME=k8s-image-autoproxy
WEBHOOK_SVC=${WEBHOOK_NAME}.${WEBHOOK_NS}.svc
K8S_OUT_CERT_FILE=./deploy/app-certs.yaml
K8S_OUT_WEBBOK_FILE=./deploy/webhooks.yaml

OUT_CERT="./ssl/${WEBHOOK_NAME}.pem"
OUT_KEY="./ssl/${WEBHOOK_NAME}.key"
   
# Create certs for our webhook 
#set -f
#mkcert \
#  --cert-file "${OUT_CERT}" \
#  --key-file "${OUT_KEY}" \
#  "${WEBHOOK_SVC}"
#mv "${OUT_CERT}" "${OUT_CERT}.bak"
#cat "${OUT_CERT}.bak" "$(mkcert -CAROOT)/rootCA.pem"  > "${OUT_CERT}"
#rm "${OUT_CERT}.bak"
#set +f
./scripts/ssl.sh $WEBHOOK_NAME $WEBHOOK_NAME

# Create certs secrets for k8s.
rm ${K8S_OUT_CERT_FILE}
kubectl -n ${WEBHOOK_NS} create secret generic \
    ${WEBHOOK_NAME}-certs \
    --from-file=key.pem=${OUT_KEY} \
    --from-file=cert.pem=${OUT_CERT}\
    --dry-run=client -o yaml > ${K8S_OUT_CERT_FILE}

# Set the CABundle on the webhook registration.
CA_BUNDLE=$(cat ${OUT_CERT} | base64 -w0)
sed "s/CA_BUNDLE/${CA_BUNDLE}/" ./deploy/webhooks.yaml.tpl > ${K8S_OUT_WEBBOK_FILE}

# Clean.
rm "${OUT_CERT}" && rm "${OUT_KEY}"

# Add note of autogenerated file.
sed -i '1i# File autogenerated by ./scripts/gen-certs.sh' ${K8S_OUT_CERT_FILE}
sed -i '1i# File autogenerated by ./scripts/gen-certs.sh' ${K8S_OUT_WEBBOK_FILE}
