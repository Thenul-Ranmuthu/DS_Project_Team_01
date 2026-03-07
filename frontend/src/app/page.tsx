import FileList from "./components/FileList";
import FileUpload from "./components/FileUpload";

export default function Home() {
  return (
    <main className="min-h-screen p-8 bg-slate-900 text-slate-100 flex flex-col items-center">
      <div className="w-full max-w-4xl space-y-8">
        <header className="text-center pb-8 border-b border-slate-700/50 pt-8">
          <h1 className="text-5xl font-extrabold tracking-tight bg-gradient-to-r from-blue-400 to-emerald-400 bg-clip-text text-transparent drop-shadow-sm">
            Distributed File Base
          </h1>
          <p className="mt-4 text-lg text-slate-400 font-medium">Powered by Go, Raft, BoltDB, and Next.js</p>
        </header>

        <section className="bg-slate-800/80 backdrop-blur-md border border-slate-700 p-8 rounded-2xl shadow-2xl hover:border-slate-600 transition-all duration-300">
          <FileUpload />
        </section>

        <section className="bg-slate-800/80 backdrop-blur-md border border-slate-700 p-8 rounded-2xl shadow-2xl hover:border-slate-600 transition-all duration-300">
          <h2 className="text-2xl font-bold mb-6 text-slate-100 flex items-center gap-2">
            <svg className="w-6 h-6 text-blue-400" fill="none" stroke="currentColor" viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 002-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10"></path></svg>
            Available Files
          </h2>
          <FileList />
        </section>
      </div>
    </main>
  );
}
