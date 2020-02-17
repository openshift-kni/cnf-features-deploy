// To compile - gcc sctpclt.c -o client -lsctp
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <unistd.h>
#include <sys/socket.h>
#include <sys/types.h>
#include <netinet/in.h>
#include <netinet/sctp.h>
#include <arpa/inet.h>
#define MAX_BUFFER 1024
#define MY_PORT_NUM 62324 /* This can be changed to suit the need and should be same in server and client */

int main(int argc, char *argv[])
{
  int connSock, in, i, ret, flags;
  struct sockaddr_in servaddr;
  struct sctp_status status;
  char buffer[MAX_BUFFER + 1];
  int datalen = 0;

  /*Get the input from user*/
  printf("Enter data to send: ");
  fgets(buffer, MAX_BUFFER, stdin);
  /* Clear the newline or carriage return from the end*/
  buffer[strcspn(buffer, "\r\n")] = 0;
  /* Sample input */
  datalen = strlen(buffer);

  connSock = socket(AF_INET, SOCK_STREAM, IPPROTO_SCTP);

  if (connSock == -1)
  {
    printf("Socket creation failed\n");
    perror("socket()");
    exit(1);
  }

  bzero((void *)&servaddr, sizeof(servaddr));
  servaddr.sin_family = AF_INET;
  servaddr.sin_port = htons(MY_PORT_NUM);
  servaddr.sin_addr.s_addr = inet_addr("127.0.0.1");

  ret = connect(connSock, (struct sockaddr *)&servaddr, sizeof(servaddr));

  if (ret == -1)
  {
    printf("Connection failed\n");
    perror("connect()");
    close(connSock);
    exit(1);
  }

  ret = sctp_sendmsg(connSock, (void *)buffer, (size_t)datalen,
                     NULL, 0, 0, 0, 0, 0, 0);
  if (ret == -1)
  {
    printf("Error in sctp_sendmsg\n");
    perror("sctp_sendmsg()");
  }
  else
    printf("Successfully sent %d bytes data to server\n", ret);

  close(connSock);

  return 0;
}
