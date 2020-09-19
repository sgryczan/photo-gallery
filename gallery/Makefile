
all: clean build sync

clean:
	rm -rf public

build:
	hugo --minify

sync:
	aws s3 sync --acl "public-read" --sse "AES256" public/ s3://photos.czan.io --delete

sync_gcloud:
	gsutil -m rsync -r public/ gs://photos.czan.io/

serve:
	hugo serve