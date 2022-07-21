package server

/*
   数据封装规则：4字节头部 + 1字节包名长度 + 包名 + protobuf数据 + 4字节adler32
   注1：4字节adler32校验的是：1字节包名长度 + 包名 + protobuf数据
   注2：4字节头部和4字节adler32采用大端序
*/

// //1.1 客户端主动向服务器请求认证。tcp连接后立即向服务器发送
// message AuthRequest {
//     string username = 1; //用户名称
//     string password = 2; //用户密码
// }
// //1.2 服务端响应客户端的认证请求
// message AuthResponse {
//     bool authorization = 1; //true为认证通过，false为认证失败
//     string interpration = 2; //如果authorization为false，则解释失败原因
// }
