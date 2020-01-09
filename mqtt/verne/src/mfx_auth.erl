-module(mfx_auth).

-behaviour(auth_on_register_hook).
-behaviour(auth_on_subscribe_hook).
-behaviour(auth_on_publish_hook).
-behaviour(on_register_hook).
-behaviour(on_client_offline_hook).
-behaviour(on_client_gone_hook).

-export([auth_on_register/5,
         auth_on_publish/6,
         auth_on_subscribe/3,
         on_register/3,
         on_client_offline/1,
         on_client_gone/1
        ]).

%% This file demonstrates the hooks you typically want to use
%% if your plugin deals with Authentication or Authorization.
%%
%% All it does is:
%%  - authenticate every user and write the log
%%  - authorize every PUBLISH and SUBSCRIBE and write it to the log
%%
%% You don't need to implement all of these hooks, just the one
%% needed for your use case.
%%
%% IMPORTANT:
%%  these hook functions run in the session context

identify(undefined) ->
    error_logger:info_msg("identify undefined", []),
    {error, undefined};
identify(Password) ->
    error_logger:info_msg("identify: ~p", [Password]),
    Token = #{value => binary_to_list(Password)},
    Worker = poolboy:checkout(grpc_pool),
    Result = gen_server:call(Worker, {identify, Token}),
    poolboy:checkin(grpc_pool, Worker),
    Result.

access(UserName, ChannelId) ->
    error_logger:info_msg("access: ~p ~p", [UserName, ChannelId]),
    AccessByIdReq = #{thingID => binary_to_list(UserName), chanID => binary_to_list(ChannelId)},
    Worker = poolboy:checkout(grpc_pool),
    Result = gen_server:call(Worker, {can_access_by_id, AccessByIdReq}),
    poolboy:checkin(grpc_pool, Worker),
    Result.

auth_on_register({_IpAddr, _Port} = Peer, {_MountPoint, _ClientId} = SubscriberId, UserName, Password, CleanSession) ->
    error_logger:info_msg("auth_on_register: ~p ~p ~p ~p ~p", [Peer, SubscriberId, UserName, Password, CleanSession]),
    %% do whatever you like with the params, all that matters
    %% is the return value of this function
    %%
    %% 1. return 'ok' -> CONNECT is authenticated
    %% 2. return 'next' -> leave it to other plugins to decide
    %% 3. return {ok, [{ModifierKey, NewVal}...]} -> CONNECT is authenticated,
    %% but we might want to set some options used throughout the client session:
    %%      - {mountpoint, NewMountPoint::string}
    %%      - {clean_session, NewCleanSession::boolean}
    %% 4. return {error, invalid_credentials} -> CONNACK_CREDENTIALS is sent
    %% 5. return {error, whatever} -> CONNACK_AUTH is sent

    case identify(Password) of
        {ok, Id} ->
            case Id of
                UserName ->
                    ok;
                _ ->
                    error_logger:info_msg("auth_on_register failed - Id: ~p, Username: ~p", [Id, UserName]),
                    {error, invalid_credentials}
            end;
        Other ->
            Other
    end.

% Erlang binary_join/2 can be found here: https://blog.kempkens.io/posts/joining-a-list-of-binaries-in-erlang/
-spec binary_join([binary()], binary()) -> binary().
binary_join([], _Sep) ->
  <<>>;
binary_join([Part], _Sep) ->
  Part;
binary_join([Head|Tail], Sep) ->
  lists:foldl(fun (Value, Acc) -> <<Acc/binary, Sep/binary, Value/binary>> end, Head, Tail).

parseTopic(Topic) when length(Topic) == 3 ->
    ChannelId = lists:nth(2, Topic),
    NatsSubject = [<<"channel">>, <<".">>, ChannelId],
    [{chanel_id, ChannelId}, {content_type, ""}, {subtopic, <<>>}, {nats_subject, NatsSubject}];
parseTopic(Topic) when length(Topic) > 3 ->
    ChannelId = lists:nth(2, Topic),
    case lists:nth(length(Topic) - 1, Topic) of
        <<"ct">> ->
            ContentType = lists:last(Topic),
            ContentType2 = re:replace(ContentType, "_", "/", [global, {return, list}]),
            ContentType3 = re:replace(ContentType2, "-", "\\+", [global, {return, list}]),
            % Subtopic is a sublist that starts at 4th element and is of full length
            % substracted by 3 elements (<<channels>>, <<ChanId>>, <<messages>>)
            % and substracted by 2 (last elements are <<ct>> and <<ContentType>>)
            Subtopic = lists:sublist(Topic, 4, length(Topic) - 3 - 2),
            Subtopic2 = binary_join(Subtopic, <<".">>),
            NatsSubject = [<<"channel">>, <<".">>, ChannelId, <<".">>, Subtopic2],
            [{chanel_id, ChannelId}, {content_type, ContentType3}, {subtopic, Subtopic2}, {nats_subject, NatsSubject}];
        _ ->
            Subtopic = lists:sublist(Topic, 4, length(Topic) - 3),
            Subtopic2 = binary_join(Subtopic, <<".">>),
            NatsSubject = [<<"channel.">>, ChannelId, <<".">>, Subtopic2],
            [{chanel_id, ChannelId}, {content_type, ""}, {subtopic, Subtopic2}, {nats_subject, NatsSubject}]
    end.

auth_on_publish(UserName, {_MountPoint, _ClientId} = SubscriberId, QoS, Topic, Payload, IsRetain) ->
    error_logger:info_msg("auth_on_publish: ~p ~p ~p ~p ~p ~p", [UserName, SubscriberId, QoS, Topic, Payload, IsRetain]),
    %% do whatever you like with the params, all that matters
    %% is the return value of this function
    %%
    %% 1. return 'ok' -> PUBLISH is authorized
    %% 2. return 'next' -> leave it to other plugins to decide
    %% 3. return {ok, NewPayload::binary} -> PUBLISH is authorized, but we changed the payload
    %% 4. return {ok, [{ModifierKey, NewVal}...]} -> PUBLISH is authorized, but we might have changed different Publish Options:
    %%     - {topic, NewTopic::string}
    %%     - {payload, NewPayload::binary}
    %%     - {qos, NewQoS::0..2}
    %%     - {retain, NewRetainFlag::boolean}
    %% 5. return {error, whatever} -> auth chain is stopped, and message is silently dropped (unless it is a Last Will message)
    %%

    % Topic is list of binaries, ex: [<<"channels">>, <<"1">>, <<"messages">>, <<"subtopic_1">>, ...]
    [{chanel_id, ChannelId}, {content_type, ContentType}, {subtopic, Subtopic}, {nats_subject, NatsSubject}] = parseTopic(Topic),
    case access(UserName, ChannelId) of
        ok ->
            Message = #{
                channel => ChannelId,
                subtopic => Subtopic,
                publisher => UserName,
                protocol => "mqtt",
                contentType => ContentType,
                payload => Payload
            },
            mfx_nats:publish(NatsSubject, message_pb:encode_msg(Message, message)),
            ok;
        Other ->
            error_logger:info_msg("Error auth: ~p", [Other]),
            Other
    end.

auth_on_subscribe(UserName, ClientId, [{Topic, _QoS}|_] = Topics) ->
    error_logger:info_msg("auth_on_subscribe: ~p ~p ~p", [UserName, ClientId, Topics]),
    %% do whatever you like with the params, all that matters
    %% is the return value of this function
    %%
    %% 1. return 'ok' -> SUBSCRIBE is authorized
    %% 2. return 'next' -> leave it to other plugins to decide
    %% 3. return {error, whatever} -> auth chain is stopped, and no SUBACK is sent

    [{chanel_id, ChannelId}, _, _, _] = parseTopic(Topic),
    access(UserName, ChannelId).

%%% Redis ES
publish_event(UserName, Type) ->
    Timestamp = os:system_time(second),
    [{_, InstanceId}] = ets:lookup(mfx_cfg, instance_id),
    KeyValuePairs = [
        "mainflux.mqtt", "*",
        "thing_id", binary_to_list(UserName),
        "timestamp", integer_to_list(Timestamp),
        "event_type", Type,
        "instance", InstanceId
    ],
    mfx_redis:publish(KeyValuePairs).

on_register(_Peer, {_Mountpoint, ClientId} = _SubscriberId, UserName) ->
    error_logger:info_msg("on_register, UserName: ~p, ClientId: ~p", [UserName, ClientId]),
    ets:insert(mfx_client_map, {ClientId, UserName}),
    publish_event(UserName, "connect").

publish_erase(ClientId) ->
    case ets:lookup(mfx_client_map, ClientId) of
        [] ->
            error_logger:info_msg("UserName for client ~p not found.", [ClientId]),
            error;
        [{ClientId, UserName}] ->
            ets:delete_object(mfx_client_map, {ClientId, UserName}),
            publish_event(UserName, "disconnect")
    end.

on_client_offline({_Mountpoint, ClientId} = _SubscriberId) ->
    error_logger:info_msg("on_client_offline, ClientId: ~p", [ClientId]),
    publish_erase(ClientId).

on_client_gone({_Mountpoint, ClientId} = _SubscriberId) ->
    error_logger:info_msg("on_client_gone, ClientId: ~p", [ClientId]),
    publish_erase(ClientId).
