@0xadcafecafe123456;

struct HelloRequest {
  name @0 :Text;
}

struct HelloResponse {
  message @0 :Text;
}

interface Greeter {
  hello @0 (req :HelloRequest) -> (resp :HelloResponse);
}
