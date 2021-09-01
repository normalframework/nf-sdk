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


var google_api_annotations_pb = require('../../google/api/annotations_pb.js')

var google_protobuf_timestamp_pb = require('google-protobuf/google/protobuf/timestamp_pb.js')

var normalgw_bacnet_bacnet_pb = require('../../normalgw/bacnet/bacnet_pb.js')
const proto = {};
proto.normalgw = {};
proto.normalgw.bacnet = require('./status_pb.js');

/**
 * @param {string} hostname
 * @param {?Object} credentials
 * @param {?Object} options
 * @constructor
 * @struct
 * @final
 */
proto.normalgw.bacnet.StatusClient =
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
proto.normalgw.bacnet.StatusPromiseClient =
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
 *   !proto.normalgw.bacnet.GetDeviceStatusRequest,
 *   !proto.normalgw.bacnet.GetDeviceStatusReply>}
 */
const methodDescriptor_Status_GetDeviceStatus = new grpc.web.MethodDescriptor(
  '/normalgw.bacnet.Status/GetDeviceStatus',
  grpc.web.MethodType.UNARY,
  proto.normalgw.bacnet.GetDeviceStatusRequest,
  proto.normalgw.bacnet.GetDeviceStatusReply,
  /**
   * @param {!proto.normalgw.bacnet.GetDeviceStatusRequest} request
   * @return {!Uint8Array}
   */
  function(request) {
    return request.serializeBinary();
  },
  proto.normalgw.bacnet.GetDeviceStatusReply.deserializeBinary
);


/**
 * @const
 * @type {!grpc.web.AbstractClientBase.MethodInfo<
 *   !proto.normalgw.bacnet.GetDeviceStatusRequest,
 *   !proto.normalgw.bacnet.GetDeviceStatusReply>}
 */
const methodInfo_Status_GetDeviceStatus = new grpc.web.AbstractClientBase.MethodInfo(
  proto.normalgw.bacnet.GetDeviceStatusReply,
  /**
   * @param {!proto.normalgw.bacnet.GetDeviceStatusRequest} request
   * @return {!Uint8Array}
   */
  function(request) {
    return request.serializeBinary();
  },
  proto.normalgw.bacnet.GetDeviceStatusReply.deserializeBinary
);


/**
 * @param {!proto.normalgw.bacnet.GetDeviceStatusRequest} request The
 *     request proto
 * @param {?Object<string, string>} metadata User defined
 *     call metadata
 * @param {function(?grpc.web.Error, ?proto.normalgw.bacnet.GetDeviceStatusReply)}
 *     callback The callback function(error, response)
 * @return {!grpc.web.ClientReadableStream<!proto.normalgw.bacnet.GetDeviceStatusReply>|undefined}
 *     The XHR Node Readable Stream
 */
proto.normalgw.bacnet.StatusClient.prototype.getDeviceStatus =
    function(request, metadata, callback) {
  return this.client_.rpcCall(this.hostname_ +
      '/normalgw.bacnet.Status/GetDeviceStatus',
      request,
      metadata || {},
      methodDescriptor_Status_GetDeviceStatus,
      callback);
};


/**
 * @param {!proto.normalgw.bacnet.GetDeviceStatusRequest} request The
 *     request proto
 * @param {?Object<string, string>} metadata User defined
 *     call metadata
 * @return {!Promise<!proto.normalgw.bacnet.GetDeviceStatusReply>}
 *     Promise that resolves to the response
 */
proto.normalgw.bacnet.StatusPromiseClient.prototype.getDeviceStatus =
    function(request, metadata) {
  return this.client_.unaryCall(this.hostname_ +
      '/normalgw.bacnet.Status/GetDeviceStatus',
      request,
      metadata || {},
      methodDescriptor_Status_GetDeviceStatus);
};


module.exports = proto.normalgw.bacnet;

