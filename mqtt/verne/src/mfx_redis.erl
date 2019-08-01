-module(mfx_redis).
-behaviour(gen_server).
-export([
    start_link/0,
    init/1,
    publish/1,
    handle_call/3,
    handle_cast/2,
    handle_info/2,
    terminate/2
]).

start_link() ->
    % Start genserver for PUB
    gen_server:start_link({local, ?MODULE}, ?MODULE, [], []).

init(_Args) ->
    error_logger:info_msg("mfx_redis genserver has started (~w)~n", [self()]),

    [{_, RedisUrl}] = ets:lookup(mfx_cfg, redis_url),
    {ok, {_, _, RedisHost, RedisPort, _, _}} = http_uri:parse(RedisUrl),
    error_logger:info_msg("mfx_redis host: ~p,  port: ~p", [RedisHost, RedisPort]),
    {ok, RedisConn} = eredis:start_link(RedisHost, RedisPort),

    ets:insert(mfx_cfg, {redis_conn, RedisConn}),
    {ok, []}.

publish(Message) ->
    gen_server:cast(?MODULE, {publish, Message}).

handle_call(Name, _From, _State) ->
    Reply = lists:flatten(io_lib:format("hello ~s from mfx_redis genserver", [Name])),
    {reply, Reply, _State}.

handle_cast({publish, Message}, _State) ->
    [{redis_conn, Conn}] = ets:lookup(mfx_cfg, redis_conn),
    error_logger:info_msg("mfx_redis genserver cast ~p ~p", [Conn, Message]),
    NewState = eredis:q(Conn, ["XADD" | Message]),
    {noreply, NewState}.

handle_info(_Info, State) ->
    {noreply, State}.

terminate(_Reason, _State) ->
    [].
