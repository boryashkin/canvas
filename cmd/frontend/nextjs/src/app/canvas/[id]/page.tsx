import { Game } from '@/components/Game';
import { notFound } from 'next/navigation';

const wsProto = process.env.WEBSOCKET_PROTOCOL ?? "ws"
const wsPort = process.env.WEBSOCKET_PORT ?? "80"

export default async function Page({
    params
  }: {
    params: { id: string }
  }) {
    console.log("canvas/[id] requested", params.id)
    if (params.id.length < 4 || params.id.length > 20) {
        notFound();
    }
    
    return (<Game gameId={params.id} websocketProto={wsProto} websocketPort={wsPort} />)
}