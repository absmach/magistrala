-module(mfx_redis).
-behaviour(gen_server).

-export([
    start_link/0,
    init/1,
    publish/1,
    handle_cast/2,
    handle_info/2,
    terminate/2
]).

-record(state, {conn}).

start_link() ->
    gen_server:start_link({local, ?MODULE}, ?MODULE, [], []).

init(_Args) ->
    error_logger:info_msg("mfx_redis genserver has started (~w)~n", [self()]),

    [{_, RedisUrl}] = ets:lookup(mfx_cfg, redis_url),
    {ok, {_, _, RedisHost, RedisPort, _, _}} = http_uri:parse(RedisUrl),
    error_logger:info_msg("mfx_redis host: ~p,  port: ~p", [RedisHost, RedisPort]),
    {ok, RedisConn} = eredis:start_link(RedisHost, RedisPort),

    {ok, #state{conn = RedisConn}}.

publish(Message) ->
    gen_server:cast(?MODULE, {publish, Message}).

handle_cast({publish, Message}, #state{conn = RedisConn} = State) ->
    [{redis_conn, Conn}] = ets:lookup(mfx_cfg, redis_conn),
    error_logger:info_msg("mfx_redis genserver cast ~p ~p", [RedisConn, Message]),
    eredis:q(Conn, ["XADD" | Message]),
    {noreply, State}.

handle_info(_Info, State) ->
    {noreply, State}.

terminate(_Reason, _State) ->
    [].
