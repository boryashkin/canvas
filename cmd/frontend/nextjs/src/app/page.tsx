
export default function Home() {
  let randCanvas = Math.random().toString(36).slice(2)
  return (
    <main className="bg-red-50 h-full p-6">
      <img src="/favicon-32x32.png" />
      <h1 className="text-lg mb-6">The canvas for sketching with friends (alpha)</h1>
      <ul className="mb-6">
        <li>Play tic-tac-toe</li>
        <li>Draw alone or with friends</li>
        <li>Doodle something</li>
      </ul>
      
      <a href={"/canvas/" + randCanvas} className="mt-4 p-4 rounded-sm bg-blue-300 text-white">Open a new canvas</a>
      <br/>
      <i className="block mt-6 text-xs">* Share a link to a canvas with friends</i>
      <br/>
      <br/>
      <br/>
      <a className="text-sm italic opacity-20" href="https://www.flaticon.com/free-icons/painting" title="painting icons">Painting icons created by Freepik - Flaticon</a>
    </main>
  )
}
