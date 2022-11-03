#!/bin/sh

docker run -d \
	--restart=unless-stopped \
	--mount type=bind,source=/root/herbiebot/feeds.db,target=/app/feeds.db \
	--mount type=bind,source=/root/herbiebot/herbie.quotes,target=/app/herbie.quotes \
	--name herbie herbie 
