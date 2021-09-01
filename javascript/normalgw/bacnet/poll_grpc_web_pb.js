/**
 * @fileoverview gRPC-Web generated client stub for normalgw.bacnet
 * @enhanceable
 * @public
 */

// GENERATED CODE -- DO NOT EDIT!


/* eslint-disable */
// @ts-nocheck



const grpc = {};
grpc.web = require('grpc-web');


var google_protobuf_timestamp_pb = require('google-protobuf/google/protobuf/timestamp_pb.js')

var google_protobuf_duration_pb = require('google-protobuf/google/protobuf/duration_pb.js')

var normalgw_bacnet_bacnet_pb = require('../../normalgw/bacnet/bacnet_pb.js')
const proto = {};
proto.normalgw = {};
proto.normalgw.bacnet = require('./poll_pb.js');

/**
 * @param {string} hostname
 * @param {?Object} credentials
 * @param {?Object} options
 * @constructor
 * @struct
 * @final
 */
proto.normalgw.bacnet.PollClient =
    function(hostname, credentials, options) {
  if (!options) options = {};
  options['format'] = 'text';

  /**
   * @private @const {!grpc.web.GrpcWebClientBase} The client
   */
  this.client_ = new grpc.web.GrpcWebClientBase(options);

  /**
   * @private @const {string} The hostname
   */
  this.hostname_ = hostname;

};


/**
 * @param {string} hostname
 * @param {?Object} credentials
 * @param {?Object} options
 * @constructor
 * @struct
 * @final
 */
proto.normalgw.bacnet.PollPromiseClient =
    function(hostname, credentials, options) {
  if (!options) options = {};
  options['format'] = 'text';

  /**
   * @private @const {!grpc.web.GrpcWebClientBase} The client
   */
  this.client_ = new grpc.web.GrpcWebClientBase(options);

  /**
   * @private @const {string} The hostname
   */
  this.hostname_ = hostname;

};


/**
 * @const
 * @type {!grpc.web.MethodDescriptor<
 *   !proto.normalgw.bacnet.GetPollablePointsRequest,
 *   !proto.normalgw.bacnet.GetPollablePointsReply>}
 */
const methodDescriptor_Poll_GetPollablePoints = new grpc.web.MethodDescriptor(
  '/normalgw.bacnet.Poll/GetPollablePoints',
  grpc.web.MethodType.UNARY,
  proto.normalgw.bacnet.GetPollablePointsRequest,
  proto.normalgw.bacnet.GetPollablePointsReply,
  /**
   * @param {!proto.normalgw.bacnet.GetPollablePointsRequest} request
   * @return {!Uint8Array}
   */
  function(request) {
    return request.serializeBinary();
  },
  proto.normalgw.bacnet.GetPollablePointsReply.deserializeBinary
);


/**
 * @const
 * @type {!grpc.web.AbstractClientBase.MethodInfo<
 *   !proto.normalgw.bacnet.GetPollablePointsRequest,
 *   !proto.normalgw.bacnet.GetPollablePointsReply>}
 */
const methodInfo_Poll_GetPollablePoints = new grpc.web.AbstractClientBase.MethodInfo(
  proto.normalgw.bacnet.GetPollablePointsReply,
  /**
   * @param {!proto.normalgw.bacnet.GetPollablePointsRequest} request
   * @return {!Uint8Array}
   */
  function(request) {
    return request.serializeBinary();
  },
  proto.normalgw.bacnet.GetPollablePointsReply.deserializeBinary
);


/**
 * @param {!proto.normalgw.bacnet.GetPollablePointsRequest} request The
 *     request proto
 * @param {?Object<string, string>} metadata User defined
 *     call metadata
 * @param {function(?grpc.web.Error, ?proto.normalgw.bacnet.GetPollablePointsReply)}
 *     callback The callback function(error, response)
 * @return {!grpc.web.ClientReadableStream<!proto.normalgw.bacnet.GetPollablePointsReply>|undefined}
 *     The XHR Node Readable Stream
 */
proto.normalgw.bacnet.PollClient.prototype.getPollablePoints =
    function(request, metadata, callback) {
  return this.client_.rpcCall(this.hostname_ +
      '/normalgw.bacnet.Poll/GetPollablePoints',
      request,
      metadata || {},
      methodDescriptor_Poll_GetPollablePoints,
      callback);
};


/**
 * @param {!proto.normalgw.bacnet.GetPollablePointsRequest} request The
 *     request proto
 * @param {?Object<string, string>} metadata User defined
 *     call metadata
 * @return {!Promise<!proto.normalgw.bacnet.GetPollablePointsReply>}
 *     Promise that resolves to the response
 */
proto.normalgw.bacnet.PollPromiseClient.prototype.getPollablePoints =
    function(request, metadata) {
  return this.client_.unaryCall(this.hostname_ +
      '/normalgw.bacnet.Poll/GetPollablePoints',
      request,
      metadata || {},
      methodDescriptor_Poll_GetPollablePoints);
};


/**
 * @const
 * @type {!grpc.web.MethodDescriptor<
 *   !proto.normalgw.bacnet.SetPollingRequest,
 *   !proto.normalgw.bacnet.SetPollingReply>}
 */
const methodDescriptor_Poll_SetPolling = new grpc.web.MethodDescriptor(
  '/normalgw.bacnet.Poll/SetPolling',
  grpc.web.MethodType.UNARY,
  proto.normalgw.bacnet.SetPollingRequest,
  proto.normalgw.bacnet.SetPollingReply,
  /**
   * @param {!proto.normalgw.bacnet.SetPollingRequest} request
   * @return {!Uint8Array}
   */
  function(request) {
    return request.serializeBinary();
  },
  proto.normalgw.bacnet.SetPollingReply.deserializeBinary
);


/**
 * @const
 * @type {!grpc.web.AbstractClientBase.MethodInfo<
 *   !proto.normalgw.bacnet.SetPollingRequest,
 *   !proto.normalgw.bacnet.SetPollingReply>}
 */
const methodInfo_Poll_SetPolling = new grpc.web.AbstractClientBase.MethodInfo(
  proto.normalgw.bacnet.SetPollingReply,
  /**
   * @param {!proto.normalgw.bacnet.SetPollingRequest} request
   * @return {!Uint8Array}
   */
  function(request) {
    return request.serializeBinary();
  },
  proto.normalgw.bacnet.SetPollingReply.deserializeBinary
);


/**
 * @param {!proto.normalgw.bacnet.SetPollingRequest} request The
 *     request proto
 * @param {?Object<string, string>} metadata User defined
 *     call metadata
 * @param {function(?grpc.web.Error, ?proto.normalgw.bacnet.SetPollingReply)}
 *     callback The callback function(error, response)
 * @return {!grpc.web.ClientReadableStream<!proto.normalgw.bacnet.SetPollingReply>|undefined}
 *     The XHR Node Readable Stream
 */
proto.normalgw.bacnet.PollClient.prototype.setPolling =
    function(request, metadata, callback) {
  return this.client_.rpcCall(this.hostname_ +
      '/normalgw.bacnet.Poll/SetPolling',
      request,
      metadata || {},
      methodDescriptor_Poll_SetPolling,
      callback);
};


/**
 * @param {!proto.normalgw.bacnet.SetPollingRequest} request The
 *     request proto
 * @param {?Object<string, string>} metadata User defined
 *     call metadata
 * @return {!Promise<!proto.normalgw.bacnet.SetPollingReply>}
 *     Promise that resolves to the response
 */
proto.normalgw.bacnet.PollPromiseClient.prototype.setPolling =
    function(request, metadata) {
  return this.client_.unaryCall(this.hostname_ +
      '/normalgw.bacnet.Poll/SetPolling',
      request,
      metadata || {},
      methodDescriptor_Poll_SetPolling);
};


module.exports = proto.normalgw.bacnet;

