This service follows the supplied tweb `appGifsManager.searchGifs`: obtain `gif_search_username` from Telegram configuration, resolve that bot and call `messages.getInlineBotResults`. It downloads public inline GIF documents into the app's own static media directory. It never sends Telegram messages or reads chats, contacts or saved GIFs.

Telegram requires an authorized **user** session for this method. The existing Telegram Bot API token can import sticker sets, but cannot authorize GIF search. Do not copy browser sessions or another application's API credentials.

1. Obtain this project's API ID/hash from https://my.telegram.org/apps and put `TELEGRAM_API_ID=…` and `TELEGRAM_API_HASH=…` into this directory's ignored `.env`.
2. In this directory, run `yarn install` and `yarn run connect`. Enter the media account's login details directly in your own terminal. The session is stored locally with mode 0600; it is never printed or bundled.
3. Run `yarn start`. The core API connects over this directory's private Unix socket. Set the same `TELEGRAM_MEDIA_SOCKET` path for both processes if relocating it.

The frontend uses CoolVibes authentication and the normal API. It never receives Telegram credentials. Session-dependent search cannot be tested live until step 2 is completed. For production, run the bridge alongside core under the same service account and supervise both processes. Cached public GIFs reside in `core/static/telegram/gifs/` and need a storage retention policy appropriate to the deployment.
