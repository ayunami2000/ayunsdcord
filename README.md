# ayunsdcord
stable diffusion ui discord bot

built for use with only one image at a time

### getting started:
Windows:
```batch
set "BOTTOKEN=YOUR_BOT_TOKEN"
go run .
```
Linux:
```sh
BOTTOKEN=YOUR_BOT_TOKEN go run .
```

---
Note that you can specify config values from both config.json and enviroment variables. The enviroment variable names are the same as the config.json names except in uppercase.  

Default config.json:
```json
{
  "bottoken": "",
  "channelids": [],
  "imagedumpchannelid": "0",
  "prefix": "sd!",
  "allowbots": false,

  "stablediffusionurl": "http://localhost:9000",
  "basicauth": "",
  "streamimageprogress": true,

  "frameurl": "",
  "framehttpbind": ":8080",
  "loadingframeurl": "https://c.tenor.com/RVvnVPK-6dcAAAAC/reload-cat.gif",

  "defaultprompt": "cat",
  "defaultnegativeprompt": "nsfw",

  "defaultwidth": 768,
  "defaultheight": 768,

  "defaultpromptstrength": 0.8,
  "defaultinferencesteps": 28,
  "defaultguidancescale": 12,
  "defaultupscaler": "",
  "defaultupscaleamount": 2,

  "denychanging": [],
  "userslist": {
    "whitelistmode": false,
    "list": []
  }
}
```