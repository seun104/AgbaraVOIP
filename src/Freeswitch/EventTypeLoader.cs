using System;
using System.Collections.Generic;
using System.Linq;
using System.Text;
using System.Reflection;


namespace Emmanuel.AgbaraVOIP.Freeswitch
{
   public class EventTypeLoader
    {
       public static void ScanForEventHandlers(Assembly assembly, Action<string, Type> foundAction)
        {
            Type eventType = typeof(EventSocket);
           Type handle = typeof(EventSocket.EventHandlers);
            foreach (Type type in assembly.GetTypes())
            {
                if (type.IsClass && type.IsSubclassOf(eventType) )
                {
                    foreach (MethodInfo m in type.GetMethods(BindingFlags.NonPublic|BindingFlags.Instance))
                    {
                        if (m.Name.ToLower().StartsWith("on_") && m.ReflectedType == handle)
                            foundAction(m.Name.Substring(3).ToLower(),type );
                    }
                }
                
            }
        }
    }
}
