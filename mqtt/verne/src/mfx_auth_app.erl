-module(mfx_auth_app).

-behaviour(application).

%% Application callbacks
-export([start/2, stop/1]).

%% ===================================================================
%% Application callbacks
%% ===================================================================

start(_StartType, _StartArgs) ->

    % Put ENV variables in ETS
    ets:new(mfx_cfg, [set, named_table, public]),

    NatsUrl = os:getenv("MF_NATS_URL", "nats://localhost:4222"),
    GrpcUrl = os:getenv("MF_THINGS_AUTH_GRPC_URL", "tcp://localhost:8183"),
    RedisUrl = os:getenv("MF_MQTT_ADAPTER_ES_URL", "tcp://localhost:6379"),
    RedisDb = os:getenv("MF_MQTT_ADAPTER_ES_DB", "0"),
    RedisPwd = os:getenv("MF_MQTT_ADAPTER_ES_PASS", ""),
    InstanceId = os:getenv("MF_MQTT_INSTANCE_ID", ""),
    PoolSize = os:getenv("MF_MQTT_VERNEMQ_GRPC_POOL_SIZE", "10"),

    ets:insert(mfx_cfg, [
        {grpc_url, GrpcUrl},
        {nats_url, NatsUrl},
        {redis_url, RedisUrl},
        {redis_db, list_to_integer(RedisDb)},
        {redis_pwd, RedisPwd},
        {instance_id, InstanceId}
    ]),

    % Also, init one ETS table for keeping the #{ClientId => Username} mapping
    ets:new(mfx_client_map, [set, named_table, public]),

    % Start the MFX Auth process
    mfx_auth_sup:start_link(list_to_integer(PoolSize)).

stop(_State) ->
    ok.


