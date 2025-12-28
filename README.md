# NYCT Feed
```
                     __ 
   ____  __  _______/ /_
  / __ \/ / / / ___/ __/
 / / / / /_/ / /__/ /_  
/_/ /_/\__, /\___/\___)  
      (____/            
```

Realtime NYC transit updates. Data provided by the [MTA](https://www.mta.info/developers).

## Local Development

You will need to [compile protocol buffers](https://protobuf.dev/getting-started/gotutorial/#compiling-protocol-buffers) when making changes to `.proto` files:

```
protoc --go_out=. --go_opt=paths=source_relative --proto_path=. internal/pb/*.proto
```
