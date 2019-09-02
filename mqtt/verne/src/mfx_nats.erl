-module(mfx_nats).
-behaviour(gen_server).
-export([
    start_link/0,
    init/1,
    publish/2,
    handle_call/3,
    handle_cast/2,
    handle_info/2,
    terminate/2,
    subscribe/1,
    loop/1
]).

-include("proto/message.hrl").

-record(state, {conn}).

start_link() ->
    % Start genserver for PUB
    gen_server:start_link({local, ?MODULE}, ?MODULE, [], []).

init(_Args) ->
    error_logger:info_msg("mfx_nats genserver has started (~w)~n", [self()]),

    [{_, NatsUrl}] = ets:lookup(mfx_cfg, nats_url),
    {ok, {_, _, NatsHost, NatsPort, _, _}} = http_uri:parse(NatsUrl),
    {ok, NatsConn} = nats:connect(list_to_binary(NatsHost), NatsPort, #{buffer_size => 10}),

    % Spawn SUB process
    spawn_link(?MODULE, subscribe, [NatsConn]),
    
    {ok, #state{conn = NatsConn}}.

publish(Subject, Message) ->
    error_logger:info_msg("mfx_nats genserver publish ~p ~p", [Subject, Message]),
    gen_server:cast(?MODULE, {publish, Subject, Message}).

% Currently unused, but kept to avoid compiler warnings (it expects handle_call/3 in the gen_server)
handle_call(Name, _From, _State) ->
    Reply = lists:flatten(io_lib:format("Hello ~s from mfx_nats genserver", [Name])),
    {reply, Reply, _State}.

handle_cast({publish, Subject, Message}, #state{conn = NatsConn} = State) ->
    error_logger:info_msg("mfx_nats genserver cast ~p ~p ~p", [Subject, NatsConn, Message]),
    nats:pub(NatsConn, Subject, #{payload => Message}),
    {noreply, State}.

handle_info(_Info, State) ->
    {noreply, State}.

terminate(_Reason, _State) ->
    [].

subscribe(NatsConn) ->
    Subject = <<"channel.>">>,
    nats:sub(NatsConn, Subject, #{queue_group => <<"mqtts">>}),
    loop(NatsConn).

loop(Conn) ->
    receive
        {Conn, ready} ->
            error_logger:info_msg("NATS ready", []),
            loop(Conn);
        {Conn, {msg, <<"teacup.control">>, _, <<"exit">>}} ->
            error_logger:info_msg("NATS received exit msg", []);
        {Conn, {msg, Subject, _ReplyTo, NatsMsg}} ->
            #{protocol := Protocol, contentType := ContentType, payload := Payload} = message:decode_msg(NatsMsg, 'mainflux.RawMessage'),
            error_logger:info_msg("Received NATS protobuf msg with payload: ~p and ContentType: ~p~n", [Payload, ContentType]),
            case Protocol of
                "mqtt" ->
                    error_logger:info_msg("Ignoring MQTT message loopback", []),
                    loop(Conn);
                _ ->
                    error_logger:info_msg("Re-publishing on MQTT broker", []),
                    ContentType2 = re:replace(ContentType, "/","_",[global,{return,list}]),
                    ContentType3 = re:replace(ContentType2, "\\+","-",[global,{return,binary}]), 
                    {_, PublishFun, {_, _}} = vmq_reg:direct_plugin_exports(?MODULE),
                    % Topic needs to be in the form of the list, like [<<"channel">>,<<"6def78cd-b441-4fd8-8680-af7e3bbea187">>]
                    Topic = case re:split(Subject, <<"\\.">>) of
                        [<<"channel">>, ChannelId] ->
                            case ContentType of
                                <<"">> ->
                                    [<<"channels">>, ChannelId, <<"messages">>];
                                _ ->
                                    [<<"channels">>, ChannelId, <<"messages">>, <<"ct">>, ContentType3]
                            end;
                        [<<"channel">>, ChannelId, Subtopic] ->
                            case ContentType of
                                <<"">> ->
                                    [<<"channels">>, ChannelId, <<"messages">>, Subtopic];
                                _ ->
                                    [<<"channels">>, ChannelId, <<"messages">>, Subtopic, <<"ct">>, ContentType3]
                            end;
                        Other ->
                            error_logger:info_msg("Could not match topic: ~p~n", [Other]),
                            error
                    end,
                    error_logger:info_msg("Subject: ~p, Topic: ~p, PublishFunction: ~p~n", [Subject, Topic, PublishFun]),
                    PublishFun(Topic, Payload, #{qos => 0, retain => false}),
                    loop(Conn)
            end;
        Other ->
            error_logger:info_msg("Received other msg: ~p~n", [Other]),
            loop(Conn)
    end.