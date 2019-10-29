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

    GrpcUrl = case os:getenv("MF_THINGS_AUTH_GRPC_URL") of
        false -> "tcp://localhost:8183";
        GrpcEnv -> GrpcEnv
    end,
    NatsUrl = case os:getenv("MF_NATS_URL") of
        false -> "nats://localhost:4222";
        NatsEnv -> NatsEnv
    end,
    RedisUrl = case os:getenv("MF_MQTT_ADAPTER_ES_URL") of
        false -> "tcp://localhost:6379";
        RedisEnv -> RedisEnv
    end,
    InstanceId = case os:getenv("MF_MQTT_INSTANCE_ID") of
        false -> "";
        InstanceEnv -> InstanceEnv
    end,
    PoolSize = case os:getenv("MF_MQTT_VERNEMQ_GRPC_POOL_SIZE") of
        false ->
            10;
        PoolSizeEnv ->
            {PoolSizeInt, _PoolSizeRest} = string:to_integer(PoolSizeEnv),
            PoolSizeInt
    end,
    
    ets:insert(mfx_cfg, [
        {grpc_url, GrpcUrl},
        {nats_url, NatsUrl},
        {redis_url, RedisUrl},
        {instance_id, InstanceId}
    ]),

    % Also, init one ETS table for keeping the #{ClientId => Username} mapping
    ets:new(mfx_client_map, [set, named_table, public]),

    % Start the MFX Auth process
    mfx_auth_sup:start_link(PoolSize).

stop(_State) ->
    ok.


