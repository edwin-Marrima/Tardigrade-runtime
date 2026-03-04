

firecracker-pack: ${OUT_DIR}/arrakis-chvapi.stamp
${OUT_DIR}/arrakis-chvapi.stamp: api/chv-api.yaml
	mkdir -p cmd/gen
	openapi-generator-cli generate -i ./.oas/firecracker.yaml -g go -o cmd/gen/firecracker --package-name ${CHV_API_GO_PACKAGE_NAME} \
    --additional-properties=withGoMod=false \
	--global-property models,supportingFiles,apis,apiTests=false
	rm -rf openapitools.json