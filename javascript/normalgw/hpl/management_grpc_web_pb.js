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
const proto = {};
proto.normalgw = {};
proto.normalgw.bacnet = require('./management_pb.js');

/**
 * @param {string} hostname
 * @param {?Object} credentials
 * @param {?Object} options
 * @constructor
 * @struct
 * @final
 */
proto.normalgw.bacnet.HplManagementClient =
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
proto.normalgw.bacnet.HplManagementPromiseClient =
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
 *   !proto.normalgw.bacnet.GetHplStatusRequest,
 *   !proto.normalgw.bacnet.GetHplStatusReply>}
 */
const methodDescriptor_HplManagement_GetHplStatus = new grpc.web.MethodDescriptor(
  '/normalgw.bacnet.HplManagement/GetHplStatus',
  grpc.web.MethodType.UNARY,
  proto.normalgw.bacnet.GetHplStatusRequest,
  proto.normalgw.bacnet.GetHplStatusReply,
  /**
   * @param {!proto.normalgw.bacnet.GetHplStatusRequest} request
   * @return {!Uint8Array}
   */
  function(request) {
    return request.serializeBinary();
  },
  proto.normalgw.bacnet.GetHplStatusReply.deserializeBinary
);


/**
 * @const
 * @type {!grpc.web.AbstractClientBase.MethodInfo<
 *   !proto.normalgw.bacnet.GetHplStatusRequest,
 *   !proto.normalgw.bacnet.GetHplStatusReply>}
 */
const methodInfo_HplManagement_GetHplStatus = new grpc.web.AbstractClientBase.MethodInfo(
  proto.normalgw.bacnet.GetHplStatusReply,
  /**
   * @param {!proto.normalgw.bacnet.GetHplStatusRequest} request
   * @return {!Uint8Array}
   */
  function(request) {
    return request.serializeBinary();
  },
  proto.normalgw.bacnet.GetHplStatusReply.deserializeBinary
);


/**
 * @param {!proto.normalgw.bacnet.GetHplStatusRequest} request The
 *     request proto
 * @param {?Object<string, string>} metadata User defined
 *     call metadata
 * @param {function(?grpc.web.Error, ?proto.normalgw.bacnet.GetHplStatusReply)}
 *     callback The callback function(error, response)
 * @return {!grpc.web.ClientReadableStream<!proto.normalgw.bacnet.GetHplStatusReply>|undefined}
 *     The XHR Node Readable Stream
 */
proto.normalgw.bacnet.HplManagementClient.prototype.getHplStatus =
    function(request, metadata, callback) {
  return this.client_.rpcCall(this.hostname_ +
      '/normalgw.bacnet.HplManagement/GetHplStatus',
      request,
      metadata || {},
      methodDescriptor_HplManagement_GetHplStatus,
      callback);
};


/**
 * @param {!proto.normalgw.bacnet.GetHplStatusRequest} request The
 *     request proto
 * @param {?Object<string, string>} metadata User defined
 *     call metadata
 * @return {!Promise<!proto.normalgw.bacnet.GetHplStatusReply>}
 *     Promise that resolves to the response
 */
proto.normalgw.bacnet.HplManagementPromiseClient.prototype.getHplStatus =
    function(request, metadata) {
  return this.client_.unaryCall(this.hostname_ +
      '/normalgw.bacnet.HplManagement/GetHplStatus',
      request,
      metadata || {},
      methodDescriptor_HplManagement_GetHplStatus);
};


module.exports = proto.normalgw.bacnet;

