# WIPCC_AliyunTTS_Go
Aliyun TTS service

### 系统设计
* 服务需要开一个`UDP Server`端口，接收其他服务发来的信令，信令中包含要合成的文本。
* 服务通过`Restful API` 向阿里云TTS服务发起HTTP请求，请求结果写入本地文件中