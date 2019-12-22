%%%-------------------------------------------------------------------
%% @doc Client module for grpc service mainflux.AuthNService.
%% @end
%%%-------------------------------------------------------------------

%% this module was generated on 2019-12-22T13:46:21+00:00 and should not be modified manually

-module(mainflux_auth_n_service_client).

-compile(export_all).
-compile(nowarn_export_all).

-include_lib("grpcbox/include/grpcbox.hrl").

-define(is_ctx(Ctx), is_tuple(Ctx) andalso element(1, Ctx) =:= ctx).

-define(SERVICE, 'mainflux.AuthNService').
-define(PROTO_MODULE, 'authn_pb').
-define(MARSHAL_FUN(T), fun(I) -> ?PROTO_MODULE:encode_msg(I, T) end).
-define(UNMARSHAL_FUN(T), fun(I) -> ?PROTO_MODULE:decode_msg(I, T) end).
-define(DEF(Input, Output, MessageType), #grpcbox_def{service=?SERVICE,
                                                      message_type=MessageType,
                                                      marshal_fun=?MARSHAL_FUN(Input),
                                                      unmarshal_fun=?UNMARSHAL_FUN(Output)}).

%% @doc Unary RPC
-spec issue(authn_pb:issue_req()) ->
    {ok, authn_pb:token(), grpcbox:metadata()} | grpcbox_stream:grpc_error_response().
issue(Input) ->
    issue(ctx:new(), Input, #{}).

-spec issue(ctx:t() | authn_pb:issue_req(), authn_pb:issue_req() | grpcbox_client:options()) ->
    {ok, authn_pb:token(), grpcbox:metadata()} | grpcbox_stream:grpc_error_response().
issue(Ctx, Input) when ?is_ctx(Ctx) ->
    issue(Ctx, Input, #{});
issue(Input, Options) ->
    issue(ctx:new(), Input, Options).

-spec issue(ctx:t(), authn_pb:issue_req(), grpcbox_client:options()) ->
    {ok, authn_pb:token(), grpcbox:metadata()} | grpcbox_stream:grpc_error_response().
issue(Ctx, Input, Options) ->
    grpcbox_client:unary(Ctx, <<"/mainflux.AuthNService/Issue">>, Input, ?DEF(issue_req, token, <<"mainflux.IssueReq">>), Options).

%% @doc Unary RPC
-spec identify(authn_pb:token()) ->
    {ok, authn_pb:user_id(), grpcbox:metadata()} | grpcbox_stream:grpc_error_response().
identify(Input) ->
    identify(ctx:new(), Input, #{}).

-spec identify(ctx:t() | authn_pb:token(), authn_pb:token() | grpcbox_client:options()) ->
    {ok, authn_pb:user_id(), grpcbox:metadata()} | grpcbox_stream:grpc_error_response().
identify(Ctx, Input) when ?is_ctx(Ctx) ->
    identify(Ctx, Input, #{});
identify(Input, Options) ->
    identify(ctx:new(), Input, Options).

-spec identify(ctx:t(), authn_pb:token(), grpcbox_client:options()) ->
    {ok, authn_pb:user_id(), grpcbox:metadata()} | grpcbox_stream:grpc_error_response().
identify(Ctx, Input, Options) ->
    grpcbox_client:unary(Ctx, <<"/mainflux.AuthNService/Identify">>, Input, ?DEF(token, user_id, <<"mainflux.Token">>), Options).

