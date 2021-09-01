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

var google_protobuf_duration_pb = require('google-protobuf/google/protobuf/duration_pb.js')

var normalgw_bacnet_bacnet_pb = require('../../normalgw/bacnet/bacnet_pb.js')

var normalgw_bacnet_bacenum_pb = require('../../normalgw/bacnet/bacenum_pb.js')
const proto = {};
proto.normalgw = {};
proto.normalgw.bacnet = require('./scan_pb.js');

/**
 * @param {string} hostname
 * @param {?Object} credentials
 * @param {?Object} options
 * @constructor
 * @struct
 * @final
 */
proto.normalgw.bacnet.ScanClient =
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
proto.normalgw.bacnet.ScanPromiseClient =
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
 *   !proto.normalgw.bacnet.GetJobRequest,
 *   !proto.normalgw.bacnet.GetJobReply>}
 */
const methodDescriptor_Scan_GetJobs = new grpc.web.MethodDescriptor(
  '/normalgw.bacnet.Scan/GetJobs',
  grpc.web.MethodType.UNARY,
  proto.normalgw.bacnet.GetJobRequest,
  proto.normalgw.bacnet.GetJobReply,
  /**
   * @param {!proto.normalgw.bacnet.GetJobRequest} request
   * @return {!Uint8Array}
   */
  function(request) {
    return request.serializeBinary();
  },
  proto.normalgw.bacnet.GetJobReply.deserializeBinary
);


/**
 * @const
 * @type {!grpc.web.AbstractClientBase.MethodInfo<
 *   !proto.normalgw.bacnet.GetJobRequest,
 *   !proto.normalgw.bacnet.GetJobReply>}
 */
const methodInfo_Scan_GetJobs = new grpc.web.AbstractClientBase.MethodInfo(
  proto.normalgw.bacnet.GetJobReply,
  /**
   * @param {!proto.normalgw.bacnet.GetJobRequest} request
   * @return {!Uint8Array}
   */
  function(request) {
    return request.serializeBinary();
  },
  proto.normalgw.bacnet.GetJobReply.deserializeBinary
);


/**
 * @param {!proto.normalgw.bacnet.GetJobRequest} request The
 *     request proto
 * @param {?Object<string, string>} metadata User defined
 *     call metadata
 * @param {function(?grpc.web.Error, ?proto.normalgw.bacnet.GetJobReply)}
 *     callback The callback function(error, response)
 * @return {!grpc.web.ClientReadableStream<!proto.normalgw.bacnet.GetJobReply>|undefined}
 *     The XHR Node Readable Stream
 */
proto.normalgw.bacnet.ScanClient.prototype.getJobs =
    function(request, metadata, callback) {
  return this.client_.rpcCall(this.hostname_ +
      '/normalgw.bacnet.Scan/GetJobs',
      request,
      metadata || {},
      methodDescriptor_Scan_GetJobs,
      callback);
};


/**
 * @param {!proto.normalgw.bacnet.GetJobRequest} request The
 *     request proto
 * @param {?Object<string, string>} metadata User defined
 *     call metadata
 * @return {!Promise<!proto.normalgw.bacnet.GetJobReply>}
 *     Promise that resolves to the response
 */
proto.normalgw.bacnet.ScanPromiseClient.prototype.getJobs =
    function(request, metadata) {
  return this.client_.unaryCall(this.hostname_ +
      '/normalgw.bacnet.Scan/GetJobs',
      request,
      metadata || {},
      methodDescriptor_Scan_GetJobs);
};


/**
 * @const
 * @type {!grpc.web.MethodDescriptor<
 *   !proto.normalgw.bacnet.StartJobRequest,
 *   !proto.normalgw.bacnet.ScanJob>}
 */
const methodDescriptor_Scan_StartJob = new grpc.web.MethodDescriptor(
  '/normalgw.bacnet.Scan/StartJob',
  grpc.web.MethodType.UNARY,
  proto.normalgw.bacnet.StartJobRequest,
  proto.normalgw.bacnet.ScanJob,
  /**
   * @param {!proto.normalgw.bacnet.StartJobRequest} request
   * @return {!Uint8Array}
   */
  function(request) {
    return request.serializeBinary();
  },
  proto.normalgw.bacnet.ScanJob.deserializeBinary
);


/**
 * @const
 * @type {!grpc.web.AbstractClientBase.MethodInfo<
 *   !proto.normalgw.bacnet.StartJobRequest,
 *   !proto.normalgw.bacnet.ScanJob>}
 */
const methodInfo_Scan_StartJob = new grpc.web.AbstractClientBase.MethodInfo(
  proto.normalgw.bacnet.ScanJob,
  /**
   * @param {!proto.normalgw.bacnet.StartJobRequest} request
   * @return {!Uint8Array}
   */
  function(request) {
    return request.serializeBinary();
  },
  proto.normalgw.bacnet.ScanJob.deserializeBinary
);


/**
 * @param {!proto.normalgw.bacnet.StartJobRequest} request The
 *     request proto
 * @param {?Object<string, string>} metadata User defined
 *     call metadata
 * @param {function(?grpc.web.Error, ?proto.normalgw.bacnet.ScanJob)}
 *     callback The callback function(error, response)
 * @return {!grpc.web.ClientReadableStream<!proto.normalgw.bacnet.ScanJob>|undefined}
 *     The XHR Node Readable Stream
 */
proto.normalgw.bacnet.ScanClient.prototype.startJob =
    function(request, metadata, callback) {
  return this.client_.rpcCall(this.hostname_ +
      '/normalgw.bacnet.Scan/StartJob',
      request,
      metadata || {},
      methodDescriptor_Scan_StartJob,
      callback);
};


/**
 * @param {!proto.normalgw.bacnet.StartJobRequest} request The
 *     request proto
 * @param {?Object<string, string>} metadata User defined
 *     call metadata
 * @return {!Promise<!proto.normalgw.bacnet.ScanJob>}
 *     Promise that resolves to the response
 */
proto.normalgw.bacnet.ScanPromiseClient.prototype.startJob =
    function(request, metadata) {
  return this.client_.unaryCall(this.hostname_ +
      '/normalgw.bacnet.Scan/StartJob',
      request,
      metadata || {},
      methodDescriptor_Scan_StartJob);
};


/**
 * @const
 * @type {!grpc.web.MethodDescriptor<
 *   !proto.normalgw.bacnet.RestartJobsRequest,
 *   !proto.normalgw.bacnet.RestartJobsReply>}
 */
const methodDescriptor_Scan_RestartJobs = new grpc.web.MethodDescriptor(
  '/normalgw.bacnet.Scan/RestartJobs',
  grpc.web.MethodType.UNARY,
  proto.normalgw.bacnet.RestartJobsRequest,
  proto.normalgw.bacnet.RestartJobsReply,
  /**
   * @param {!proto.normalgw.bacnet.RestartJobsRequest} request
   * @return {!Uint8Array}
   */
  function(request) {
    return request.serializeBinary();
  },
  proto.normalgw.bacnet.RestartJobsReply.deserializeBinary
);


/**
 * @const
 * @type {!grpc.web.AbstractClientBase.MethodInfo<
 *   !proto.normalgw.bacnet.RestartJobsRequest,
 *   !proto.normalgw.bacnet.RestartJobsReply>}
 */
const methodInfo_Scan_RestartJobs = new grpc.web.AbstractClientBase.MethodInfo(
  proto.normalgw.bacnet.RestartJobsReply,
  /**
   * @param {!proto.normalgw.bacnet.RestartJobsRequest} request
   * @return {!Uint8Array}
   */
  function(request) {
    return request.serializeBinary();
  },
  proto.normalgw.bacnet.RestartJobsReply.deserializeBinary
);


/**
 * @param {!proto.normalgw.bacnet.RestartJobsRequest} request The
 *     request proto
 * @param {?Object<string, string>} metadata User defined
 *     call metadata
 * @param {function(?grpc.web.Error, ?proto.normalgw.bacnet.RestartJobsReply)}
 *     callback The callback function(error, response)
 * @return {!grpc.web.ClientReadableStream<!proto.normalgw.bacnet.RestartJobsReply>|undefined}
 *     The XHR Node Readable Stream
 */
proto.normalgw.bacnet.ScanClient.prototype.restartJobs =
    function(request, metadata, callback) {
  return this.client_.rpcCall(this.hostname_ +
      '/normalgw.bacnet.Scan/RestartJobs',
      request,
      metadata || {},
      methodDescriptor_Scan_RestartJobs,
      callback);
};


/**
 * @param {!proto.normalgw.bacnet.RestartJobsRequest} request The
 *     request proto
 * @param {?Object<string, string>} metadata User defined
 *     call metadata
 * @return {!Promise<!proto.normalgw.bacnet.RestartJobsReply>}
 *     Promise that resolves to the response
 */
proto.normalgw.bacnet.ScanPromiseClient.prototype.restartJobs =
    function(request, metadata) {
  return this.client_.unaryCall(this.hostname_ +
      '/normalgw.bacnet.Scan/RestartJobs',
      request,
      metadata || {},
      methodDescriptor_Scan_RestartJobs);
};


/**
 * @const
 * @type {!grpc.web.MethodDescriptor<
 *   !proto.normalgw.bacnet.ImportJobRequest,
 *   !proto.normalgw.bacnet.ImportJobReply>}
 */
const methodDescriptor_Scan_ImportJob = new grpc.web.MethodDescriptor(
  '/normalgw.bacnet.Scan/ImportJob',
  grpc.web.MethodType.UNARY,
  proto.normalgw.bacnet.ImportJobRequest,
  proto.normalgw.bacnet.ImportJobReply,
  /**
   * @param {!proto.normalgw.bacnet.ImportJobRequest} request
   * @return {!Uint8Array}
   */
  function(request) {
    return request.serializeBinary();
  },
  proto.normalgw.bacnet.ImportJobReply.deserializeBinary
);


/**
 * @const
 * @type {!grpc.web.AbstractClientBase.MethodInfo<
 *   !proto.normalgw.bacnet.ImportJobRequest,
 *   !proto.normalgw.bacnet.ImportJobReply>}
 */
const methodInfo_Scan_ImportJob = new grpc.web.AbstractClientBase.MethodInfo(
  proto.normalgw.bacnet.ImportJobReply,
  /**
   * @param {!proto.normalgw.bacnet.ImportJobRequest} request
   * @return {!Uint8Array}
   */
  function(request) {
    return request.serializeBinary();
  },
  proto.normalgw.bacnet.ImportJobReply.deserializeBinary
);


/**
 * @param {!proto.normalgw.bacnet.ImportJobRequest} request The
 *     request proto
 * @param {?Object<string, string>} metadata User defined
 *     call metadata
 * @param {function(?grpc.web.Error, ?proto.normalgw.bacnet.ImportJobReply)}
 *     callback The callback function(error, response)
 * @return {!grpc.web.ClientReadableStream<!proto.normalgw.bacnet.ImportJobReply>|undefined}
 *     The XHR Node Readable Stream
 */
proto.normalgw.bacnet.ScanClient.prototype.importJob =
    function(request, metadata, callback) {
  return this.client_.rpcCall(this.hostname_ +
      '/normalgw.bacnet.Scan/ImportJob',
      request,
      metadata || {},
      methodDescriptor_Scan_ImportJob,
      callback);
};


/**
 * @param {!proto.normalgw.bacnet.ImportJobRequest} request The
 *     request proto
 * @param {?Object<string, string>} metadata User defined
 *     call metadata
 * @return {!Promise<!proto.normalgw.bacnet.ImportJobReply>}
 *     Promise that resolves to the response
 */
proto.normalgw.bacnet.ScanPromiseClient.prototype.importJob =
    function(request, metadata) {
  return this.client_.unaryCall(this.hostname_ +
      '/normalgw.bacnet.Scan/ImportJob',
      request,
      metadata || {},
      methodDescriptor_Scan_ImportJob);
};


/**
 * @const
 * @type {!grpc.web.MethodDescriptor<
 *   !proto.normalgw.bacnet.ObserveJobUpdatesRequest,
 *   !proto.normalgw.bacnet.ObserveJobUpdatesReply>}
 */
const methodDescriptor_Scan_ObserveJobUpdates = new grpc.web.MethodDescriptor(
  '/normalgw.bacnet.Scan/ObserveJobUpdates',
  grpc.web.MethodType.SERVER_STREAMING,
  proto.normalgw.bacnet.ObserveJobUpdatesRequest,
  proto.normalgw.bacnet.ObserveJobUpdatesReply,
  /**
   * @param {!proto.normalgw.bacnet.ObserveJobUpdatesRequest} request
   * @return {!Uint8Array}
   */
  function(request) {
    return request.serializeBinary();
  },
  proto.normalgw.bacnet.ObserveJobUpdatesReply.deserializeBinary
);


/**
 * @const
 * @type {!grpc.web.AbstractClientBase.MethodInfo<
 *   !proto.normalgw.bacnet.ObserveJobUpdatesRequest,
 *   !proto.normalgw.bacnet.ObserveJobUpdatesReply>}
 */
const methodInfo_Scan_ObserveJobUpdates = new grpc.web.AbstractClientBase.MethodInfo(
  proto.normalgw.bacnet.ObserveJobUpdatesReply,
  /**
   * @param {!proto.normalgw.bacnet.ObserveJobUpdatesRequest} request
   * @return {!Uint8Array}
   */
  function(request) {
    return request.serializeBinary();
  },
  proto.normalgw.bacnet.ObserveJobUpdatesReply.deserializeBinary
);


/**
 * @param {!proto.normalgw.bacnet.ObserveJobUpdatesRequest} request The request proto
 * @param {?Object<string, string>} metadata User defined
 *     call metadata
 * @return {!grpc.web.ClientReadableStream<!proto.normalgw.bacnet.ObserveJobUpdatesReply>}
 *     The XHR Node Readable Stream
 */
proto.normalgw.bacnet.ScanClient.prototype.observeJobUpdates =
    function(request, metadata) {
  return this.client_.serverStreaming(this.hostname_ +
      '/normalgw.bacnet.Scan/ObserveJobUpdates',
      request,
      metadata || {},
      methodDescriptor_Scan_ObserveJobUpdates);
};


/**
 * @param {!proto.normalgw.bacnet.ObserveJobUpdatesRequest} request The request proto
 * @param {?Object<string, string>} metadata User defined
 *     call metadata
 * @return {!grpc.web.ClientReadableStream<!proto.normalgw.bacnet.ObserveJobUpdatesReply>}
 *     The XHR Node Readable Stream
 */
proto.normalgw.bacnet.ScanPromiseClient.prototype.observeJobUpdates =
    function(request, metadata) {
  return this.client_.serverStreaming(this.hostname_ +
      '/normalgw.bacnet.Scan/ObserveJobUpdates',
      request,
      metadata || {},
      methodDescriptor_Scan_ObserveJobUpdates);
};


/**
 * @const
 * @type {!grpc.web.MethodDescriptor<
 *   !proto.normalgw.bacnet.GetScanDeltaRequest,
 *   !proto.normalgw.bacnet.GetScanDeltaReply>}
 */
const methodDescriptor_Scan_GetScanDeltas = new grpc.web.MethodDescriptor(
  '/normalgw.bacnet.Scan/GetScanDeltas',
  grpc.web.MethodType.SERVER_STREAMING,
  proto.normalgw.bacnet.GetScanDeltaRequest,
  proto.normalgw.bacnet.GetScanDeltaReply,
  /**
   * @param {!proto.normalgw.bacnet.GetScanDeltaRequest} request
   * @return {!Uint8Array}
   */
  function(request) {
    return request.serializeBinary();
  },
  proto.normalgw.bacnet.GetScanDeltaReply.deserializeBinary
);


/**
 * @const
 * @type {!grpc.web.AbstractClientBase.MethodInfo<
 *   !proto.normalgw.bacnet.GetScanDeltaRequest,
 *   !proto.normalgw.bacnet.GetScanDeltaReply>}
 */
const methodInfo_Scan_GetScanDeltas = new grpc.web.AbstractClientBase.MethodInfo(
  proto.normalgw.bacnet.GetScanDeltaReply,
  /**
   * @param {!proto.normalgw.bacnet.GetScanDeltaRequest} request
   * @return {!Uint8Array}
   */
  function(request) {
    return request.serializeBinary();
  },
  proto.normalgw.bacnet.GetScanDeltaReply.deserializeBinary
);


/**
 * @param {!proto.normalgw.bacnet.GetScanDeltaRequest} request The request proto
 * @param {?Object<string, string>} metadata User defined
 *     call metadata
 * @return {!grpc.web.ClientReadableStream<!proto.normalgw.bacnet.GetScanDeltaReply>}
 *     The XHR Node Readable Stream
 */
proto.normalgw.bacnet.ScanClient.prototype.getScanDeltas =
    function(request, metadata) {
  return this.client_.serverStreaming(this.hostname_ +
      '/normalgw.bacnet.Scan/GetScanDeltas',
      request,
      metadata || {},
      methodDescriptor_Scan_GetScanDeltas);
};


/**
 * @param {!proto.normalgw.bacnet.GetScanDeltaRequest} request The request proto
 * @param {?Object<string, string>} metadata User defined
 *     call metadata
 * @return {!grpc.web.ClientReadableStream<!proto.normalgw.bacnet.GetScanDeltaReply>}
 *     The XHR Node Readable Stream
 */
proto.normalgw.bacnet.ScanPromiseClient.prototype.getScanDeltas =
    function(request, metadata) {
  return this.client_.serverStreaming(this.hostname_ +
      '/normalgw.bacnet.Scan/GetScanDeltas',
      request,
      metadata || {},
      methodDescriptor_Scan_GetScanDeltas);
};


module.exports = proto.normalgw.bacnet;

