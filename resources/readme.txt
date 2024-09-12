Please note:
The image file is only necessary to create the video file

To create a video from image please use one of the following commands libx264 will be used in both command for encoding:
ffmpeg -loop 1 -i stream-limit.jpg -c:v libx264 -t 1 -pix_fmt yuv420p -vf scale=1920:1080  stream-limit.ts
cvlc --no-audio --loop --sout "#transcode{vcodec=h264,vb=1024,scale=1,width=1920,height=1080,acodec=none,venc=x264{preset=ultrafast}}:standard{access=file,mux=ts,dst=stream-limit.ts}" stream-limit.jpg