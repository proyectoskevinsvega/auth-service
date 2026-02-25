package grpc

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/peer"
)

// ClientIdentityKey is the context key for the client identity extracted from the certificate
type ClientIdentityKey struct{}

// UnaryIdentityInterceptor extracts the Common Name (CN) from the client's mTLS certificate
func UnaryIdentityInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		p, ok := peer.FromContext(ctx)
		if ok && p.AuthInfo != nil {
			if tlsInfo, ok := p.AuthInfo.(credentials.TLSInfo); ok {
				if len(tlsInfo.State.PeerCertificates) > 0 {
					// Get the first certificate (the client cert)
					cert := tlsInfo.State.PeerCertificates[0]
					clientName := cert.Subject.CommonName

					// Inject identity into context
					ctx = context.WithValue(ctx, ClientIdentityKey{}, clientName)
				}
			}
		}

		return handler(ctx, req)
	}
}

// GetClientIdentity retrieves the client identity from the context
func GetClientIdentity(ctx context.Context) string {
	if identity, ok := ctx.Value(ClientIdentityKey{}).(string); ok {
		return identity
	}
	return "unknown"
}
