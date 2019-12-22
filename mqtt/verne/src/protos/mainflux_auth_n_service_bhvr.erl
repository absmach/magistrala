%%%-------------------------------------------------------------------
%% @doc Behaviour to implement for grpc service mainflux.AuthNService.
%% @end
%%%-------------------------------------------------------------------

%% this module was generated on 2019-12-22T13:46:21+00:00 and should not be modified manually

-module(mainflux_auth_n_service_bhvr).

%% @doc Unary RPC
-callback issue(ctx:ctx(), authn_pb:issue_req()) ->
    {ok, authn_pb:token(), ctx:ctx()} | grpcbox_stream:grpc_error_response().

%% @doc Unary RPC
-callback identify(ctx:ctx(), authn_pb:token()) ->
    {ok, authn_pb:user_id(), ctx:ctx()} | grpcbox_stream:grpc_error_response().

