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

    AuthUrl = case os:getenv("MF_THINGS_AUTH_HTTP_URL") of
        false -> "http://localhost:8989";
        AuthEnv -> AuthEnv
    end,
    NatsUrl = case os:getenv("MF_NATS_URL") of
        false -> "nats://localhost:4222";
        NatsEnv -> NatsEnv
    end,
    RedisUrl = case os:getenv("MF_MQTT_ADAPTER_ES_URL") of
        false -> "tcp://localhost:6379";
        RedisEnv -> RedisEnv
    end,

    ets:insert(mfx_cfg, [
        {auth_url, AuthUrl},
        {nats_url, NatsUrl},
        {redis_url, RedisUrl}
    ]),

    % Also, init one ETS table for keeping the #{ClientId => Username} mapping
    ets:new(mfx_client_map, [set, named_table, public]),

    % Start Hackney
    application:ensure_all_started(hackney),

    % Start the process
    mfx_auth_sup:start_link().

stop(_State) ->
    ok.
