-module(mfx_grpc).
-behaviour(gen_server).

-export([
    start_link/0,
    start_link/1,
    init/1,
    handle_call/3,
    handle_cast/2,
    handle_info/2,
    terminate/2
]).

-record(state, {channel}).

init(_Args) ->
    error_logger:info_msg("mfx_grpc genserver has started (~w)~n", [self()]),
    [{_, GrpcUrl}] = ets:lookup(mfx_cfg, grpc_url),
    {ok, {_, _, GrpcHost, GrpcPort, _, _}} = http_uri:parse(GrpcUrl),
    error_logger:info_msg("gRPC host: ~p,  port: ~p", [GrpcHost, GrpcPort]),
    Channel = list_to_atom(pid_to_list(self())),
    grpcbox_channel_sup:start_child(Channel, [{http, GrpcHost, GrpcPort, []}], #{}),
    {ok, #state{channel = Channel}}.

start_link() ->
    gen_server:start_link({local, ?MODULE}, ?MODULE, [], []).

start_link(Args) ->
    gen_server:start_link(?MODULE, Args, []).

handle_call({identify, Message}, _From, #state{channel = Channel} = State) ->
    error_logger:info_msg("mfx_grpc message: ~p, channel: ~p", [Message, Channel]),
    {ok, Resp, HeadersAndTrailers} = mainflux_things_service_client:identify(Message, #{channel => Channel}),
    case maps:get(<<":status">>, maps:get(headers, HeadersAndTrailers)) of
        <<"200">> ->
            {reply, {ok, maps:get(value, Resp)}, State};
        ErrorStatus ->
            {reply, {error, ErrorStatus}, State}
    end;

handle_call({can_access_by_id, Message}, _From, #state{channel = Channel} = State) ->
    error_logger:info_msg("mfx_grpc message: ~p, channel: ~p", [Message, Channel]),
    {ok, _, HeadersAndTrailers} = mainflux_things_service_client:can_access_by_id(Message, #{channel => Channel}),
    error_logger:info_msg("mfx_grpc can_access_by_id() HeadersAndTrailers: ~p", [HeadersAndTrailers]),
    case maps:get(<<":status">>, maps:get(headers, HeadersAndTrailers)) of
        <<"200">> ->
            {reply, ok, State};
        ErrorStatus ->
            {reply, {error, ErrorStatus}, State}
    end.

handle_cast(_Request, State) ->
    {noreply, State}.

handle_info(_Info, State) ->
    {noreply, State}.

terminate(Reason, #state{channel = Channel} =State) ->
    grpcbox_channel:stop(Channel),
    {stop, Reason, State}.
