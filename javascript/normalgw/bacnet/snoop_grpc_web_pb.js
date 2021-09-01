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

const proto = {};
proto.normalgw = {};
proto.normalgw.bacnet = require('./snoop_pb.js');

/**
 * @param {string} hostname
 * @param {?Object} credentials
 * @param {?Object} options
 * @constructor
 * @struct
 * @final
 */
proto.normalgw.bacnet.SnoopClient =
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
proto.normalgw.bacnet.SnoopPromiseClient =
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
 *   !proto.normalgw.bacnet.GetSnoopRequest,
 *   !proto.normalgw.bacnet.GetSnoopReply>}
 */
const methodDescriptor_Snoop_GetSnoop = new grpc.web.MethodDescriptor(
  '/normalgw.bacnet.Snoop/GetSnoop',
  grpc.web.MethodType.UNARY,
  proto.normalgw.bacnet.GetSnoopRequest,
  proto.normalgw.bacnet.GetSnoopReply,
  /**
   * @param {!proto.normalgw.bacnet.GetSnoopRequest} request
   * @return {!Uint8Array}
   */
  function(request) {
    return request.serializeBinary();
  },
  proto.normalgw.bacnet.GetSnoopReply.deserializeBinary
);


/**
 * @const
 * @type {!grpc.web.AbstractClientBase.MethodInfo<
 *   !proto.normalgw.bacnet.GetSnoopRequest,
 *   !proto.normalgw.bacnet.GetSnoopReply>}
 */
const methodInfo_Snoop_GetSnoop = new grpc.web.AbstractClientBase.MethodInfo(
  proto.normalgw.bacnet.GetSnoopReply,
  /**
   * @param {!proto.normalgw.bacnet.GetSnoopRequest} request
   * @return {!Uint8Array}
   */
  function(request) {
    return request.serializeBinary();
  },
  proto.normalgw.bacnet.GetSnoopReply.deserializeBinary
);


/**
 * @param {!proto.normalgw.bacnet.GetSnoopRequest} request The
 *     request proto
 * @param {?Object<string, string>} metadata User defined
 *     call metadata
 * @param {function(?grpc.web.Error, ?proto.normalgw.bacnet.GetSnoopReply)}
 *     callback The callback function(error, response)
 * @return {!grpc.web.ClientReadableStream<!proto.normalgw.bacnet.GetSnoopReply>|undefined}
 *     The XHR Node Readable Stream
 */
proto.normalgw.bacnet.SnoopClient.prototype.getSnoop =
    function(request, metadata, callback) {
  return this.client_.rpcCall(this.hostname_ +
      '/normalgw.bacnet.Snoop/GetSnoop',
      request,
      metadata || {},
      methodDescriptor_Snoop_GetSnoop,
      callback);
};


/**
 * @param {!proto.normalgw.bacnet.GetSnoopRequest} request The
 *     request proto
 * @param {?Object<string, string>} metadata User defined
 *     call metadata
 * @return {!Promise<!proto.normalgw.bacnet.GetSnoopReply>}
 *     Promise that resolves to the response
 */
proto.normalgw.bacnet.SnoopPromiseClient.prototype.getSnoop =
    function(request, metadata) {
  return this.client_.unaryCall(this.hostname_ +
      '/normalgw.bacnet.Snoop/GetSnoop',
      request,
      metadata || {},
      methodDescriptor_Snoop_GetSnoop);
};


module.exports = proto.normalgw.bacnet;

