# Changelog

## 0.1.0 (2026-03-07)


### Features

* add additional info to room ([3ff36b4](https://github.com/choffmann/chat-room/commit/3ff36b44d0d3ca5db3af9740e5f601909441dbaf))
* add endpoint to get user by id ([28aa756](https://github.com/choffmann/chat-room/commit/28aa756f62d576919313b2d7680acf6250fdec11))
* add get all rooms endpoint ([ef8869d](https://github.com/choffmann/chat-room/commit/ef8869d4b3b72005c24865874e3453434d8d9e85))
* add id to user and add simple logging ([fee7717](https://github.com/choffmann/chat-room/commit/fee77175ab0761250c8bcadcfe0d5b91bf4a576d))
* add info endpoint ([71499ca](https://github.com/choffmann/chat-room/commit/71499ca54eece34094e13441b9f1b788a81c86da))
* add leave message from system when client disconnect ([9423328](https://github.com/choffmann/chat-room/commit/9423328ab51fccc980a74382a50b8d70ae1fd64f))
* add release-please and improve build info ([d12ec10](https://github.com/choffmann/chat-room/commit/d12ec10444a0fac9f205bbc529822d93b74e8e17))
* change message type (WIP) ([5d41f58](https://github.com/choffmann/chat-room/commit/5d41f58ec473a957ba670e2e0f748800c4b5e0aa))
* clean room after timeout ([6634eb3](https://github.com/choffmann/chat-room/commit/6634eb39302f6b39396b76a9334bc322adcdec3f))
* create websocket server ([3e22369](https://github.com/choffmann/chat-room/commit/3e2236918b1605099b60c574a541602450bd0d6a))
* display how many user are online in a room ([b2a678d](https://github.com/choffmann/chat-room/commit/b2a678d6fdb68b075a229078c83febbec010a082))
* do not safe if message is empty ([6c4f8f1](https://github.com/choffmann/chat-room/commit/6c4f8f11b221f357c56acb8ec61c592eb9cd9f7e))
* get user by room and add user endpoint ([210b7e7](https://github.com/choffmann/chat-room/commit/210b7e7b37d6c329ec62f300d37b14cb83beb85b))
* improve logging ([c533cf2](https://github.com/choffmann/chat-room/commit/c533cf26ab6e414b28f962b848890629f8a4530e))
* init repo ([2192ada](https://github.com/choffmann/chat-room/commit/2192adacf0fcf20658183a305abce3d0d4739c41))
* patch and put rooms additional info ([be0a633](https://github.com/choffmann/chat-room/commit/be0a633d54b00c2a849b06bd5c0fb57e7e70a751))
* save messages and add new messages endpoints ([2668d43](https://github.com/choffmann/chat-room/commit/2668d4341b2185ef0b6361a7ddf8f24d243877d3))
* send user info to connected client when query parameyer userInfo is set ([19a96c9](https://github.com/choffmann/chat-room/commit/19a96c9226455c6df97de7d91d8c1bbc8ae1252e))
* update messages and add user names ([a844db1](https://github.com/choffmann/chat-room/commit/a844db13bc1190361b4223bf7af336ffb474200a))
* use 4 digit room id ([7e2d554](https://github.com/choffmann/chat-room/commit/7e2d5547afaa315990d2a034a694359d994c901f))


### Bug Fixes

* add content type header in /rooms/{roomId} ([99ed53b](https://github.com/choffmann/chat-room/commit/99ed53b14986babd6f8edb08a910632d67190940))
* add PATCH to cors ([d1b743d](https://github.com/choffmann/chat-room/commit/d1b743d8335946bd55dd51c232f8f5f9e57bd523))
* enable cors origin ([88ded56](https://github.com/choffmann/chat-room/commit/88ded56a23263e5a5ca1a3caef7831757662369c))
* send join message to all clients and not to joined client ([61153f9](https://github.com/choffmann/chat-room/commit/61153f900277ccce519fb5b96073fe59884d3d75))
* user count in room response ([383e2e6](https://github.com/choffmann/chat-room/commit/383e2e60c45a0b93db19df49dc0387cff3d92160))


### Code Refactoring

* restructure to idiomatic Go project layoutt ([0099402](https://github.com/choffmann/chat-room/commit/0099402a249851694bb61ad3d437fa5a536a00a4))
