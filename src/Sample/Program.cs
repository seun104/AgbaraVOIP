using System;
using System.Collections.Generic;
using System.Linq;
using System.Text;
using System.Diagnostics;
using System.Threading;
using System.Threading.Tasks;

namespace Sample
{
    class Program
    {
        static void Main(string[] args)
        {
            //sample application
            Thread.Sleep(5000);//sleep for a while
            Task.Factory.StartNew(() => Sample.AgbaraAPI.ApiSample.Sample1());

            Console.ReadLine();
        }
    }
}
