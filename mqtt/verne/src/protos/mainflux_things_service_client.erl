%%%-------------------------------------------------------------------
%% @doc Client module for grpc service mainflux.ThingsService.
%% @end
%%%-------------------------------------------------------------------

%% this module was generated on 2019-11-04T09:57:59+00:00 and should not be modified manually

-module(mainflux_things_service_client).

-compile(export_all).
-compile(nowarn_export_all).

-include_lib("grpcbox/include/grpcbox.hrl").

-define(is_ctx(Ctx), is_tuple(Ctx) andalso element(1, Ctx) =:= ctx).

-define(SERVICE, 'mainflux.ThingsService').
-define(PROTO_MODULE, 'internal_pb').
-define(MARSHAL_FUN(T), fun(I) -> ?PROTO_MODULE:encode_msg(I, T) end).
-define(UNMARSHAL_FUN(T), fun(I) -> ?PROTO_MODULE:decode_msg(I, T) end).
-define(DEF(Input, Output, MessageType), #grpcbox_def{service=?SERVICE,
                                                      message_type=MessageType,
                                                      marshal_fun=?MARSHAL_FUN(Input),
                                                      unmarshal_fun=?UNMARSHAL_FUN(Output)}).

%% @doc Unary RPC
-spec can_access_by_key(internal_pb:access_by_key_req()) ->
    {ok, internal_pb:thing_id(), grpcbox:metadata()} | grpcbox_stream:grpc_error_response().
can_access_by_key(Input) ->
    can_access_by_key(ctx:new(), Input, #{}).

-spec can_access_by_key(ctx:t() | internal_pb:access_by_key_req(), internal_pb:access_by_key_req() | grpcbox_client:options()) ->
    {ok, internal_pb:thing_id(), grpcbox:metadata()} | grpcbox_stream:grpc_error_response().
can_access_by_key(Ctx, Input) when ?is_ctx(Ctx) ->
    can_access_by_key(Ctx, Input, #{});
can_access_by_key(Input, Options) ->
    can_access_by_key(ctx:new(), Input, Options).

-spec can_access_by_key(ctx:t(), internal_pb:access_by_key_req(), grpcbox_client:options()) ->
    {ok, internal_pb:thing_id(), grpcbox:metadata()} | grpcbox_stream:grpc_error_response().
can_access_by_key(Ctx, Input, Options) ->
    grpcbox_client:unary(Ctx, <<"/mainflux.ThingsService/CanAccessByKey">>, Input, ?DEF(access_by_key_req, thing_id, <<"mainflux.AccessByKeyReq">>), Options).

%% @doc Unary RPC
-spec can_access_by_id(internal_pb:access_by_id_req()) ->
    {ok, internal_pb:empty(), grpcbox:metadata()} | grpcbox_stream:grpc_error_response().
can_access_by_id(Input) ->
    can_access_by_id(ctx:new(), Input, #{}).

-spec can_access_by_id(ctx:t() | internal_pb:access_by_id_req(), internal_pb:access_by_id_req() | grpcbox_client:options()) ->
    {ok, internal_pb:empty(), grpcbox:metadata()} | grpcbox_stream:grpc_error_response().
can_access_by_id(Ctx, Input) when ?is_ctx(Ctx) ->
    can_access_by_id(Ctx, Input, #{});
can_access_by_id(Input, Options) ->
    can_access_by_id(ctx:new(), Input, Options).

-spec can_access_by_id(ctx:t(), internal_pb:access_by_id_req(), grpcbox_client:options()) ->
    {ok, internal_pb:empty(), grpcbox:metadata()} | grpcbox_stream:grpc_error_response().
can_access_by_id(Ctx, Input, Options) ->
    grpcbox_client:unary(Ctx, <<"/mainflux.ThingsService/CanAccessByID">>, Input, ?DEF(access_by_id_req, empty, <<"mainflux.AccessByIDReq">>), Options).

%% @doc Unary RPC
-spec identify(internal_pb:token()) ->
    {ok, internal_pb:thing_id(), grpcbox:metadata()} | grpcbox_stream:grpc_error_response().
identify(Input) ->
    identify(ctx:new(), Input, #{}).

-spec identify(ctx:t() | internal_pb:token(), internal_pb:token() | grpcbox_client:options()) ->
    {ok, internal_pb:thing_id(), grpcbox:metadata()} | grpcbox_stream:grpc_error_response().
identify(Ctx, Input) when ?is_ctx(Ctx) ->
    identify(Ctx, Input, #{});
identify(Input, Options) ->
    identify(ctx:new(), Input, Options).

-spec identify(ctx:t(), internal_pb:token(), grpcbox_client:options()) ->
    {ok, internal_pb:thing_id(), grpcbox:metadata()} | grpcbox_stream:grpc_error_response().
identify(Ctx, Input, Options) ->
    grpcbox_client:unary(Ctx, <<"/mainflux.ThingsService/Identify">>, Input, ?DEF(token, thing_id, <<"mainflux.Token">>), Options).

